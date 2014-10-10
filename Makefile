depends:
	go get github.com/garyburd/redigo/redis
	go get github.com/pelletier/go-toml
	go get github.com/go-martini/martini
	go get github.com/martini-contrib/render
	go get github.com/blackjack/syslog
	
build:
	go build -v -o lilliput src/main.go
