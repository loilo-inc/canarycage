package awsiface

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// coverage cheat: always use MustLoadConfig to avoid error handling repetition
func MustLoadConfig(ctx context.Context, opts ...func(*config.LoadOptions) error) aws.Config {
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return cfg
}
