all: _all
mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface/mock_interface.go: ./vendor/github.com/aws/aws-sdk-go/service/ecs/ecsiface/interface.go
	mkdir -p mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface
	mockgen -source ./vendor/github.com/aws/aws-sdk-go/service/ecs/ecsiface/interface.go > mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface/mock_interface.go
mocks/github.com/aws/aws-sdk-go/service/ec2/ec2iface/mock_interface.go: ./vendor/github.com/aws/aws-sdk-go/service/ec2/ec2iface/interface.go
	mkdir -p mocks/github.com/aws/aws-sdk-go/service/ec2/ec2iface
	mockgen -source ./vendor/github.com/aws/aws-sdk-go/service/ec2/ec2iface/interface.go > mocks/github.com/aws/aws-sdk-go/service/ec2/ec2iface/mock_interface.go
mocks/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface/mock_interface.go: ./vendor/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface/interface.go
	mkdir -p mocks/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface
	mockgen -source ./vendor/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface/interface.go > mocks/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface/mock_interface.go
_all:  mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface/mock_interface.go mocks/github.com/aws/aws-sdk-go/service/ec2/ec2iface/mock_interface.go mocks/github.com/aws/aws-sdk-go/service/elbv2/elbv2iface/mock_interface.go
clean:
	rm -rf mocks/*
