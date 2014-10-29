// The MIT License (MIT)

// Copyright (c) 2014 Jade E Services Pvt. Ltd.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package liliput

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/blackjack/syslog"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/gorelic"
	"github.com/martini-contrib/render"
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
	ERROR_UNAUTHORIZED            = 6
)

type Entity struct {
	Tiny    string `json:"url"`
	Err     int    `json:"error_code"`
	Message string `json:"message"`
	Token   string `json:"_"`
	Url     string `json:"_"`
}

// Creating redis server pool
func NewPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
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

// called on post request to create tiny url
func TinyUrl(req *http.Request, r render.Render, pool *redis.Pool) {
	entity := &Entity{}
	e, err := entity.Save(req.FormValue("url"), pool)
	if err != nil {
		r.JSON(400, e)
		return
	}

	r.JSON(200, e)
	return
}

func (entity *Entity) Save(iaddress string, pool *redis.Pool) (*Entity, error) {
	resource := strings.TrimSpace(iaddress)
	if len(resource) == 0 {
		entity.Err = ERROR_NO_URL
		entity.Message = "url must be provided in request."
		err := errors.New(entity.Message)
		return entity, err
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
		entity.Err = ERROR_INVALID_DOMAIN
		entity.Message = "Invalid domain in url"
		err := errors.New(entity.Message)
		return entity, err
	}

	expression := regexp.MustCompile(EMAIL_REGEX)
	if expression.MatchString(resource) == false {
		entity.Err = ERROR_INVALID_URL
		entity.Message = "Invalid url"
		err := errors.New(entity.Message)
		return entity, err
	}
	entity.Url = iaddress
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
			entity.Token = id
			break
		}
	}

	if entity.Token == "" {
		syslog.Critf("Error: failed to generate token")
		entity.Err = ERROR_TOKEN_GENERATION_FAILED
		entity.Message = "Faild to generate token please try again."
		err := errors.New(entity.Message)
		return entity, err
	}

	reply, err := db.Do("SET", entity.Token, entity.Url)
	if err == nil && reply != "OK" {
		syslog.Critf("Error: %s", err)
		entity.Err = ERROR_FAILED_TO_SAVE
		entity.Message = "Invalid Redis response"
		err := errors.New(entity.Message)
		return entity, err
	}

	entity.Err = SUCCESS
	entity.Tiny = Get("lilliput.domain", "").(string) + entity.Token
	syslog.Critf("Tiny url from %s to %s", entity.Url, entity.Tiny)
	return entity, nil
}

func (entity *Entity) Retrieve(pool *redis.Pool) error {
	db := pool.Get()
	defer db.Close()
	url, err := redis.String(db.Do("GET", entity.Token))
	if err == nil {
		entity.Url = url
	}
	return err
}

// redirect go goes from here
func Redirect(params martini.Params, r render.Render, pool *redis.Pool) {
	entity := &Entity{}
	entity.Token = params["token"]
	err := entity.Retrieve(pool)
	if err != nil {
		syslog.Critf("Error: Token not found %s", params["token"])
		r.HTML(404, "404", nil)
	} else {
		syslog.Critf("Redirect from %s to %s", Get("lilliput.domain", "").(string)+params["token"], entity.Url)
		r.Redirect(entity.Url, 301)
	}
}

func Start() {
	fmt.Println("Starting Liliput..")
	syslog.Openlog("lilliput", syslog.LOG_PID, syslog.LOG_USER)

	m := martini.Classic()
	// render home page
	m.Use(render.Renderer(render.Options{
		Directory:  "static",
		Extensions: []string{".html"},
		Charset:    "UTF-8",
	}))

	gorelic.InitNewrelicAgent(Get("newrelic.license", "").(string), Get("lilliput.domain", "").(string), true)
	m.Use(gorelic.Handler)

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
