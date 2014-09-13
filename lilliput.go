package liliput

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pilu/base62"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

var Id WebServiceResponse

type WebServiceResponse struct {
	Id        int
	MacId     string
	RegiserId string
}

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

func TinyUrl(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	params := mux.Vars(req)
	fmt.Println(params)
	encoded := base62.Encode(21122)
	tiny := make(map[string]string)
	tiny["url"] = Get("lilliput.domain", "").(string) + encoded
	enc, _ := json.Marshal(tiny)
	resp.Write(enc)
	return
}

func Start() {
	fmt.Println("Starting Liliput..")
	r := mux.NewRouter()
	r.HandleFunc("/liliput", TinyUrl).Methods("POST")
	http.Handle("/", r)
	fmt.Println("Started...")
	port := fmt.Sprintf(":%v", Get("lilliput.port", ""))
	http.ListenAndServe(port, nil)
}
