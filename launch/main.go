package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pilu/base62"
	"net/http"
)

func TinyUrl(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	params := mux.Vars(req)
	fmt.Println(params)
	encoded := base62.Encode(21122)
	tiny := make(map[string]string)
	tiny["url"] = "http://liliput.com/" + encoded
	enc, _ := json.Marshal(tiny)
	resp.Write(enc)
	return
}

func main() {
	fmt.Println("Starting Liliput..")
	r := mux.NewRouter()
	r.HandleFunc("/liliput", TinyUrl).Methods("POST")
	http.Handle("/", r)
	fmt.Println("Started...")
	http.ListenAndServe(":8989", nil)
}
