package main

import (
	"os"
	"net/http"
	"fmt"
	"time"
)

func main() {
	mode := os.Args[1]
	switch mode {
	case "healthy":
		HealthyServer()
	case "unhealthy":
		UnHealthyServer()
	case "up-but-buggy":
		BuggyServer()
	case "up-but-slow":
		SlowServer()
	}
	http.ListenAndServe(":8000", nil)
}

func HealthyServer() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "🐤")
	})
	http.HandleFunc("/health_check", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	})
}
func UnHealthyServer() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "🐤")
	})

}

func BuggyServer() {
	i := 0
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if i++; i%2 == 0 {
			writer.WriteHeader(500)
		} else {
			fmt.Fprintf(writer, "🐤")
		}
	})
	http.HandleFunc("/health_check", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	})
}

func SlowServer() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		d := time.Duration(2) * time.Second
		timer := time.NewTimer(d)
		<-timer.C
		fmt.Fprintf(writer, "🐤")
	})
	http.HandleFunc("/health_check", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	})
}
