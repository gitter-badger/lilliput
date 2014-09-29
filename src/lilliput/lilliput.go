package liliput

import (
	"base62"
	"fmt"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/garyburd/redigo/redis"
	"github.com/go-martini/martini"
	"net/http"
	"regexp"
	"time"
)

type Data struct {
	Url     string `json:"url"`
	Err     bool   `json:"err"`
	Message string `json:"message"`
	OrgUrl  string
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

func TinyUrl(req *http.Request, r render.Render, pool *redis.Pool) {
	data := &Data{OrgUrl: req.FormValue("url")}
	expression := regexp.MustCompile(`(http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?`)
	if expression.MatchString(data.OrgUrl) {
		data.Save(pool)
	} else {
		data.Err = true
		data.Message = "Provide valid url, url must be with prepend with http:// or https://"
	}
	r.JSON(200, data)
	return
}

func (data *Data) Save(pool *redis.Pool) {
	db := pool.Get()
	defer db.Close()
	n, err := redis.Int(db.Do("INCR", "id"))
	db.Do("SET", n, data.OrgUrl)
	if err != nil {
		fmt.Println(err)
		data.Err = true
		data.Message = "Faild to generate please try again."
	} else {
		data.Err = false
		encoded := base62.Encode(n)
		data.Url = Get("lilliput.domain", "").(string) + encoded
	}
}

func (data *Data) Retrieve(pool *redis.Pool, token string) error {
	decoded := base62.Decode(token)
	db := pool.Get()
	defer db.Close()
	url, err := redis.String(db.Do("GET", decoded))
	if err == nil {
		data.OrgUrl = url
	}
	return err
}

func Redirect(params martini.Params, r render.Render, pool *redis.Pool) {
	data := &Data{}
	err := data.Retrieve(pool, params["token"])
	if err != nil {
		r.HTML(404, "404", nil)
	} else {
		r.Redirect(data.OrgUrl, 301)
	}
}

func Start() {
	fmt.Println("Starting Liliput..")
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
	m.Map(newPool(server))

	port := fmt.Sprintf(":%v", Get("lilliput.port", ""))
	fmt.Println("Started on " + port + "...")
	http.Handle("/", m)
	http.ListenAndServe(port, nil)
	m.Run()
}
