FROM golang:alpine
COPY main.go /go/github.com/loilo-inc/canarycage/test-container/
ENTRYPOINT ["go", "run", "/go/github.com/loilo-inc/canarycage/test-container/main.go"]
CMD["healthy"]