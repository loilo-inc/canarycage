module github.com/loilo-inc/canarycage/v5

go 1.26.0

toolchain go1.27.4

require (
	github.com/aws/aws-sdk-go-v2 v1.47.5
	github.com/aws/aws-sdk-go-v2/config v1.33.12
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.343.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.67.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.103.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.63.6
	github.com/google/go-github/v62 v62.0.0
	github.com/google/uuid v1.6.0
	github.com/jarcoal/httpmock v1.4.2
	github.com/stretchr/testify v1.12.1
	github.com/urfave/cli/v3 v3.14.0
	go.uber.org/mock v0.6.0
	golang.org/x/sync v0.24.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.12.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

require (
	github.com/Masterminds/semver/v3 v3.5.0
	github.com/aws/aws-sdk-go-v2/credentials v1.20.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.8 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.39.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.44.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.51.6 // indirect
	github.com/aws/smithy-go v1.28.7
	github.com/loilo-inc/logos/v2 v2.1.0
)
