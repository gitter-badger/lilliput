package liliput

import (
	"crypto/rand"
	"fmt"
	"github.com/blackjack/syslog"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	ALPHANUM                      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	EMAIL_REGEX                   = `(http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?`
	SUCCESS                       = 0
	ERROR_NO_URL                  = 1
	ERROR_INVALID_DOMAIN          = 2
	ERROR_INVALID_URL             = 3
	ERROR_FAILED_TO_SAVE          = 4
	ERROR_TOKEN_GENERATION_FAILED = 5
)

type Data struct {
	Tiny    string `json:"url"`
	Err     int    `json:"error_code"`
	Message string `json:"message"`
	Token   string `json:"_"`
	Url     string `json:"_"`
}

func NewPool(server string) *redis.Pool {
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

func TinyUrl(req *http.Request, r render.Render, pool *redis.Pool) {
	resource := strings.TrimSpace(req.FormValue("url"))
	data := &Data{}

	if len(resource) == 0 {
		data.Err = ERROR_NO_URL
		data.Message = "url must be provided in request."
		r.JSON(200, data)
		return
	}

	u, _ := url.Parse(resource)
	domains := Get("lilliput.alloweddomain", nil).([]interface{})
	flag := true
	for _, d := range domains {
		if d.(string) == u.Host {
			flag = false
		}
	}

	if flag {
		data.Err = ERROR_INVALID_DOMAIN
		data.Message = "Invalid domain in url"
		r.JSON(200, data)
		return
	}

	expression := regexp.MustCompile(EMAIL_REGEX)
	if expression.MatchString(resource) == false {
		data.Err = ERROR_INVALID_URL
		data.Message = "Invalid url"
		r.JSON(200, data)
		return
	}

	bytes := make([]byte, 7)
	db := pool.Get()
	defer db.Close()

	for i := 0; i < 5; i++ {
		rand.Read(bytes)
		for i, b := range bytes {
			bytes[i] = ALPHANUM[b%byte(len(ALPHANUM))]
		}
		id := string(bytes)
		if exists, _ := redis.Bool(db.Do("EXISTS", id)); !exists {
			data.Token = id
			break
		}
	}

	if data.Token == "" {
		syslog.Critf("Error: failed to generate token")
		data.Err = ERROR_TOKEN_GENERATION_FAILED
		data.Message = "Faild to generate token please try again."
		r.JSON(200, data)
		return
	}

	data.Save(pool)
	r.JSON(200, data)
	return
}

func (data *Data) Save(pool *redis.Pool) {
	db := pool.Get()
	defer db.Close()

	_, err := db.Do("SET", data.Token, data.Url)
	if err != nil {
		syslog.Critf("Error: %s", err)
		data.Err = ERROR_FAILED_TO_SAVE
		data.Message = "Faild to generate please try again."
	} else {
		data.Err = SUCCESS
		data.Url = Get("lilliput.domain", "").(string) + data.Token
		syslog.Critf("Tiny url from %s to %s", data.Url, data.Tiny)
	}
}

func (data *Data) Retrieve(pool *redis.Pool) error {
	db := pool.Get()
	defer db.Close()
	url, err := redis.String(db.Do("GET", data.Token))
	if err == nil {
		data.Url = url
	}
	return err
}

func Redirect(params martini.Params, r render.Render, pool *redis.Pool) {
	data := &Data{}
	data.Token = params["token"]
	err := data.Retrieve(pool)
	if err != nil {
		syslog.Critf("Error: Token not found %s", params["token"])
		r.HTML(404, "404", nil)
	} else {
		syslog.Critf("Redirect from %s to %s", Get("lilliput.domain", "").(string)+params["token"], data.Url)
		r.Redirect(data.Url, 301)
	}
}

func Start() {
	fmt.Println("Starting Liliput..")
	syslog.Openlog("lilliput", syslog.LOG_PID, syslog.LOG_USER)

	m := martini.Classic()
	m.Use(render.Renderer(render.Options{
		Directory:  "static",
		Extensions: []string{".html"},
		Charset:    "UTF-8",
	}))

	m.Get("/:token", Redirect)
	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", nil)
	})
	m.Post("/", TinyUrl)
	server := fmt.Sprintf("%s:%s",
		Get("redis.server", ""),
		Get("redis.port", ""))
	m.Map(NewPool(server))

	port := fmt.Sprintf(":%v", Get("lilliput.port", ""))
	fmt.Println("Started on " + port + "...")
	http.Handle("/", m)
	http.ListenAndServe(port, nil)
	m.Run()
}
