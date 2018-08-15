FROM golang:alpine as build
WORKDIR /build
COPY main.go /build/
RUN go build -o http-server

FROM alpine:3.8
COPY --from=build /build/http-server /workspace/
EXPOSE 8000
ENTRYPOINT ["/workspace/http-server"]
CMD ["healthy"]