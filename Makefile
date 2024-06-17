MOCKGEN := go run github.com/golang/mock/mockgen@v1.6.0
.PHONY: test
test:
	go test ./... -coverprofile=coverage.txt -covermode=count
test-container:
	docker build -t canarycage/test-container test-container
push-test-container: test-container
	docker tag canarycage/test-container loilodev/http-server:latest
	docker push loilodev/http-server:latest
version:
	go run cli/cage/main.go -v | cut -f 3 -d ' '
mocks: mocks/mock_awsiface/iface.go mocks/mock_cage/iface.go mocks/mock_upgrade/upgrade.go
mocks/mock_awsiface/iface.go: awsiface/iface.go
	$(MOCKGEN) -source=./awsiface/iface.go > mocks/mock_awsiface/iface.go
mocks/mock_cage/iface.go: cage.go
	$(MOCKGEN) -source=./cage.go > mocks/mock_cage/cage.go
mocks/mock_upgrade/upgrade.go: cli/cage/upgrade/upgrade.go
	$(MOCKGEN) -source=./cli/cage/upgrade/upgrade.go > mocks/mock_upgrade/upgrade.go
.PHONY: mocks
