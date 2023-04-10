MOCKGEN := go run github.com/golang/mock/mockgen@v1.6.0
.PHONY: test
test:
	go test -coverprofile=coverage.txt -covermode=count
test-container:
	docker build -t canarycage/test-container test-container
push-test-container: test-container
	docker tag canarycage/test-container loilodev/http-server:latest
	docker push loilodev/http-server:latest
version:
	go run cli/cage/main.go -v | cut -f 3 -d ' '

mocks:
	$(MOCKGEN) github.com/aws/aws-sdk-go/service/ec2/ec2iface EC2API > mocks/github.com/aws/aws-sdk-go/service/ec2/ec2iface/mock_interface.go
	$(MOCKGEN) github.com/aws/aws-sdk-go/service/ecs/ecsiface ECSAPI > mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface/mock_interface.go
	$(MOCKGEN) github.com/aws/aws-sdk-go/service/elbv2/elbv2iface ELBV2API > mocks/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface/mock_interface.go

.PHONY: mocks
