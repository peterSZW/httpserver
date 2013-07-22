package main

import (
	"fmt"
	"net/http"
	//"os"
)

var ch chan int

type User struct {
	ID       string
	Msg      string
	isOnline bool
	ch       chan int
}

var UserDB map[string]User

func Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>Hello %s!</h1>", r.URL.Path[1:])
}

func getmsg(w http.ResponseWriter, r *http.Request) {

	num := <-ch

	fmt.Fprintf(w, "<h1>getmsg %d!</h1>", num)
}

func setmsg(w http.ResponseWriter, r *http.Request) {
	ch <- 1
	fmt.Fprintf(w, "<h1>setmsg %d!</h1>", 1)
}

func main() {
	ch = make(chan int, 1000)
	http.HandleFunc("/", Handler)
	http.HandleFunc("/getmsg", getmsg)
	http.HandleFunc("/setmsg", setmsg)
	http.ListenAndServe(":80", nil)
}

//http.HandleFunc("/get", getEnv)
//
//func getEnv(writer http.ResponseWriter, req *http.Request) {
//	env := os.Environ()
//	writer.Write([]byte("<h1>Envirment</h1><br>"))
//	for _, v := range env {
//		writer.Write([]byte(v + "<br>"))
//	}
//	writer.Write([]byte("<br>"))
//}
