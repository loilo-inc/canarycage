#!/usr/bin/env bash
docker run --rm --privileged \
    -v ${PWD}:/go/src/github.com/loilo-inc/canarycage \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /go/src/github.com/loilo-inc/canarycage \
    -e GITHUB_TOKEN=${GITHUB_TOKEN} \
    goreleaser/goreleaser release --rm-dist