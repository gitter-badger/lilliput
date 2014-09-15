package liliput

import (
	"base62"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	_ "path"
	"regexp"
	"strings"
	"time"
)

var Id WebServiceResponse

type WebServiceResponse struct {
	Id         int
	MacId      string
	RegisterId string
}

type Data struct {
	Url string
}

var data Data

type Response struct {
	Url     string `json:"url"`
	Err     bool   `json:"err"`
	Message string `json:"message"`
}

var response Response

var pool *redis.Pool

func init() {
	interfaces, _ := net.InterfaceByName(Get("lilliput.interface", "").(string))
	url := Get("lilliput.webservice", "").(string) + interfaces.HardwareAddr.String()
	fmt.Println("Registring to webservice..")
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(contents))
	json.Unmarshal([]byte(contents), &Id)
	fmt.Println("Registration complete.")
	// initialize pool
	InitPool()
}

func newPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			c.Do("SELECT", Get("redis.dbname", "").(int64))
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func InitPool() {
	val := fmt.Sprintf("%s:%s",
		Get("redis.server", ""),
		Get("redis.port", ""))
	pool = newPool(val)
}

func TinyUrl(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	fmt.Println(req.FormValue("url"))
	data.Url = req.FormValue("url")
	expression := regexp.MustCompile(`(http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?`)
	fmt.Println(expression.MatchString(data.Url))

	if expression.MatchString(data.Url) {
		StoreData()
		response.Err = false
	} else {
		response.Err = true
		response.Message = "Provide valid url,url must be with prepend with http:// or https://"
	}

	r, _ := json.Marshal(response)
	resp.Write(r)
	return
}

func StoreData() {
	db := pool.Get()
	defer db.Close()
	n, err := redis.Int(db.Do("INCR", "id"))
	db.Do("SET", n, data.Url)
	if err != nil {
		fmt.Println(err)
		response.Err = true
		response.Message = "Faild to generate please try again."
	} else {
		encoded := base62.Encode(n)
		response.Url = Get("lilliput.domain", "").(string) + encoded + Id.RegisterId
		fmt.Println(response.Url)
	}
}

func Redirect(resp http.ResponseWriter, req *http.Request) {
	fmt.Println("Redirecting from " + Id.RegisterId)
	params := mux.Vars(req)
	fmt.Println(params["liliput"])
	encoded := strings.TrimSuffix(params["liliput"], Id.RegisterId)
	fmt.Println(encoded)
	decoded := base62.Decode(encoded)
	fmt.Println(decoded)
	db := pool.Get()
	defer db.Close()
	val, err := redis.String(db.Do("GET", decoded))
	if err != nil {
		fmt.Println(err)
		http.Redirect(resp, req, "/notfound/", 404)
		return
	}
	http.Redirect(resp, req, val, 301)
}

func Start() {
	fmt.Println("Starting Liliput..")
	r := mux.NewRouter()
	r.HandleFunc("/", TinyUrl).Methods("POST")
	r.HandleFunc("/", RenderHome).Methods("GET")
	r.HandleFunc("/{liliput}", Redirect).Methods("GET")
	r.HandleFunc("/notfound/", RenderNotFound).Methods("GET")
	http.Handle("/", r)
	port := fmt.Sprintf(":%v", Get("lilliput.port", ""))
	fmt.Println("Started on " + port + "...")
	http.ListenAndServe(port, nil)
}

func RenderHome(resp http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(resp, err.Error(), status)
		}
	}()
	fpath := "./static/index.html"

	resp.Header().Set("Content-Type", "text/html")
	if err = servefile(resp, fpath); nil != err {
		status = http.StatusInternalServerError
		return
	}
}

func servefile(res http.ResponseWriter, fpath string) (err error) {
	outfile, err := os.OpenFile(fpath, os.O_RDONLY, 0x0444)
	if nil != err {
		return
	}

	// 32k buffer copy
	written, err := io.Copy(res, outfile)
	if nil != err {
		return
	}

	fmt.Println("served file:", outfile.Name(), ";length:", written)
	return
}

func RenderNotFound(resp http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(resp, err.Error(), status)
		}
	}()
	fpath := "./static/404.html"

	// http.Error(resp, err.Error(), 404)
	// http.Error(resp, "", http.StatusNotFound)
	resp.Header().Set("Content-Type", "text/html")
	if err = servefile(resp, fpath); nil != err {
		status = http.StatusInternalServerError
		return
	}
}
