test:
	go test -coverprofile=coverage.txt -covermode=count
test-container:
	docker build -t canarycage/test-container test-container
push-test-container: test-container
	docker tag canarycage/test-container loilodev/http-server:latest
	docker push loilodev/http-server:latest
release:
	docker run --rm --privileged \
		-v ${PWD}:/go/src/github.com/loilo-inc/canarycage \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ${HOME}/.aws:/root/.aws \
		-w /go/src/github.com/loilo-inc/canarycage \
		-e GITHUB_TOKEN=`lake decrypt -f .github_token.enc` \
		goreleaser/goreleaser release --rm-dist
version:
	go run cli/cage/main.go -v | cut -f 3 -d ' '
