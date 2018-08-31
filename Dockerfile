FROM golang:1.10 as dep
ENV WORK_DIR /go/src/github.com/loilo-inc/canarycage
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR ${WORK_DIR}
COPY *.go Gopkg.lock Gopkg.toml ${WORK_DIR}/
COPY test ${WORK_DIR}/test
COPY mock ${WORK_DIR}/mock
RUN dep ensure

FROM golang:1.10-alpine3.8
ENV WORK_DIR /go/src/github.com/loilo-inc/canarycage
RUN apk update --no-cache \
    && apk add git bash curl make \
    && go get -u github.com/keroxp/shake \
    && go get -u github.com/golang/dep/cmd/dep \
    && go get -u github.com/goreleaser/goreleaser
WORKDIR ${WORK_DIR}
COPY . ${WORK_DIR}
COPY --from=dep ${WORK_DIR}/vendor ${WORK_DIR}/vendor