package main

import (
	"os"
	"net/http"
	"fmt"
	"time"
	"log"
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
	case "up-but-exit":
		log.Fatalf("😱")
	case "up-and-exit":
		HealthyServer()
		timer := time.NewTimer(time.Duration(1) * time.Minute)
		go func() {
			<-timer.C
			log.Fatalf("😈")
		}()
	}
	port := "8000"
	if o, ok := os.LookupEnv("PORT"); ok {
		port = o
	}
	http.ListenAndServe(":"+port, nil)
	log.Printf("http-server is now runnin as %s mode", mode)
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
	http.HandleFunc("/health_check", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(500)
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
