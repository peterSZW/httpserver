package main

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
)

var cnt int
var ch chan bool

func getmsg(w http.ResponseWriter, r *http.Request) {
	msg := "0"

	if cnt > 0 {
		msg = fmt.Sprintf("%d", cnt)
	}

	if cnt > 0 {
		ch <- true
		runtime.Gosched()
	}

	cnt++

	defer func() {
		cnt--
	}()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10e9)
		timeout <- true
	}()

	select {
	case <-timeout:
		fmt.Fprintf(w, "<h1>timeout %s</h1>", msg)
	case <-ch:
		fmt.Fprintf(w, "<h1>break </h1>")

	}

}

func main() {
	cnt = 0
	ch = make(chan bool, 10)

	http.HandleFunc("/getmsg", getmsg)

	http.ListenAndServe(":80", nil)
}
