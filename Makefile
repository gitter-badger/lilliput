depends:
	go get github.com/garyburd/redigo/redis
	go get github.com/gorilla/mux
	go get github.com/pelletier/go-toml

build:
	go build -v -o lilliput src/main.go
