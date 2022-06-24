package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

func server1(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(rw, "this is server 1 on port 1111")
}

func server2(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(rw, "this is server 2 on port 2222")
}

func server3(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(rw, "this is server 3 on port 3333")
}

func server4(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(rw, "this is server 4 on port 4444")
}

func main() {
	listener1, err := net.Listen("tcp", ":1111")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	listener2, err := net.Listen("tcp", ":2222")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	listener3, err := net.Listen("tcp", ":3333")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	listener4, err := net.Listen("tcp", ":4444")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	go http.Serve(listener1, http.HandlerFunc(server1))
	go http.Serve(listener2, http.HandlerFunc(server2))
	go http.Serve(listener3, http.HandlerFunc(server3))
	go http.Serve(listener4, http.HandlerFunc(server4))

	select {}
}
