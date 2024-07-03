package task

import "context"

type Task interface {
	Start(ctx context.Context) error
	Wait(ctx context.Context) error
	Stop(ctx context.Context) error
}
