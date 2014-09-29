depends:
	go get github.com/garyburd/redigo/redis
	go get github.com/pelletier/go-toml
	go get github.com/go-martini/martini
	
build:
	go build -v -o lilliput src/main.go
