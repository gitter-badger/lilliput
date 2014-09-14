depends:
	go get github.com/garyburd/redigo/redis
	go get github.com/gorilla/mux

build:
	go build -v -o lilliput src/main.go