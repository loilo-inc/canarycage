# go.modのバージョンを使うと、missing go.sum entry for module providing package...エラーが出る
MOCKGEN := go run go.uber.org/mock/mockgen@v0.6.0
.PHONY: test
test:
	go test ./... -coverprofile=coverage.txt -covermode=count
test/cli:
	go test ./cli/... -coverprofile=coverage-cli.txt -covermode=count
test-container:
	docker build -t canarycage/test-container test-container
push-test-container: test-container
	docker tag canarycage/test-container loilodev/http-server:latest
	docker push loilodev/http-server:latest
version:
	go run cli/cage/main.go -v | cut -f 3 -d ' '
mocks: go.sum \
	mocks/mock_awsiface/iface.go \
	mocks/mock_types/iface.go \
	mocks/mock_upgrade/upgrade.go \
	mocks/mock_scan/scanner.go \
	mocks/mock_task/task.go \
	mocks/mock_taskset/taskset.go \
	mocks/mock_task/factory.go \
	mocks/mock_rollout/executor.go \
	mocks/mock_logger/logger.go
mocks/mock_awsiface/iface.go: awsiface/iface.go
	$(MOCKGEN) -source=./awsiface/iface.go > mocks/mock_awsiface/iface.go
mocks/mock_types/iface.go: types/iface.go
	$(MOCKGEN) -source=./types/iface.go > mocks/mock_types/iface.go
mocks/mock_upgrade/upgrade.go: cli/cage/upgrade/upgrade.go
	$(MOCKGEN) -source=./cli/cage/upgrade/upgrade.go > mocks/mock_upgrade/upgrade.go
mocks/mock_scan/scanner.go: cli/cage/audit/scanner.go
	$(MOCKGEN) -source=./cli/cage/audit/scanner.go > mocks/mock_audit/scanner.go
mocks/mock_task/task.go: task/task.go
	$(MOCKGEN) -source=./task/task.go > mocks/mock_task/task.go
mocks/mock_taskset/taskset.go: taskset/taskset.go
	$(MOCKGEN) -source=./taskset/taskset.go > mocks/mock_taskset/taskset.go
mocks/mock_task/factory.go: task/factory.go
	$(MOCKGEN) -source=./task/factory.go > mocks/mock_task/factory.go
mocks/mock_rollout/executor.go: rollout/executor.go
	$(MOCKGEN) -source=./rollout/executor.go > mocks/mock_rollout/executor.go
mocks/mock_logger/logger.go: logger/logger.go
	$(MOCKGEN) -source=./logger/logger.go > mocks/mock_logger/logger.go
.PHONY: mocks
