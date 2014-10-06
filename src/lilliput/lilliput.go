package liliput

import (
	"crypto/rand"
	"fmt"
	"github.com/blackjack/syslog"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type Data struct {
	Url     string `json:"url"`
	Err     bool   `json:"err"`
	Message string `json:"message"`
	token   string
	OrgUrl  string
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
	data := &Data{OrgUrl: strings.TrimSpace(req.FormValue("url"))}
	expression := regexp.MustCompile(`(http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?`)
	if expression.MatchString(data.OrgUrl) {
		data.Save(pool)
	} else {
		syslog.Critf("Invalid url %s", data.OrgUrl)
		data.Err = true
		data.Message = "Provide valid url, url must be with prepend with http:// or https://"
	}

	r.JSON(200, data)
	return
}

func (data *Data) Save(pool *redis.Pool) {
	db := pool.Get()
	defer db.Close()
	bytes := make([]byte, 7)

	for i := 0; i < 5; i++ {
		rand.Read(bytes)
		for i, b := range bytes {
			bytes[i] = alphanum[b%byte(len(alphanum))]
		}
		id := string(bytes)
		if exists, _ := redis.Bool(db.Do("EXISTS", id)); !exists {
			data.token = id
			break
		}
	}

	if data.token == "" {
		syslog.Critf("Error: failed to generate token")
		data.Err = true
		data.Message = "Faild to generate token please try again."
	} else {
		_, err := db.Do("SET", data.token, data.OrgUrl)
		if err != nil {
			syslog.Critf("Error: %s", err)
			data.Err = true
			data.Message = "Faild to generate please try again."
		} else {
			data.Err = false
			data.Url = Get("lilliput.domain", "").(string) + data.token
		}
		syslog.Critf("Tiny url from %s to %s", data.OrgUrl, data.Url)
	}
}

func (data *Data) Retrieve(pool *redis.Pool) error {
	db := pool.Get()
	defer db.Close()
	url, err := redis.String(db.Do("GET", data.token))
	if err == nil {
		data.OrgUrl = url
	}
	return err
}

func Redirect(params martini.Params, r render.Render, pool *redis.Pool) {
	data := &Data{}
	data.token = params["token"]
	err := data.Retrieve(pool)
	if err != nil {
		syslog.Critf("Error: Token not found %s", params["token"])
		r.HTML(404, "404", nil)
	} else {
		syslog.Critf("Redirect from %s to %s", Get("lilliput.domain", "").(string)+params["token"], data.OrgUrl)
		r.Redirect(data.OrgUrl, 301)
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
