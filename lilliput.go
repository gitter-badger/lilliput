package liliput

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/pilu/base62"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
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

var resource redis.Conn

func init() {
	interfaces, _ := net.InterfaceByName("eth0")
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
}

func Db() redis.Conn {
	if resource == nil {
		val := fmt.Sprintf("%s:%s",
			Get("redis.server", ""),
			Get("redis.port", ""))
		fmt.Println(val)
		var err error
		resource, err = redis.Dial("tcp", val)
		resource.Do("SELECT", 0)
		if err != nil {
			panic(err)
		}
	}
	return resource
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
	db := Db()
	n, err := redis.Int(db.Do("INCR", data.Url))
	if err != nil {
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
	db := Db()
	// val, err := redis.String(db.Do("GET", string(decoded)))
	val, err := db.Do("GET", decoded)
	fmt.Println(err)
	fmt.Println(val)
	//remove last character from params & decode, fetch url from db
	// http.Redirect(resp, req, "http://google.com", 301)
}

func Start() {
	fmt.Println("Starting Liliput..")
	r := mux.NewRouter()
	r.HandleFunc("/", TinyUrl).Methods("POST")
	r.HandleFunc("/{liliput}", Redirect).Methods("GET")
	http.Handle("/", r)
	port := fmt.Sprintf(":%v", Get("lilliput.port", ""))
	fmt.Println("Started on " + port + "...")
	http.ListenAndServe(port, nil)
}
