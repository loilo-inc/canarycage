package commands

import (
	"context"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/urfave/cli"
)

type CageCommands interface {
	Up() cli.Command
	RollOut() cli.Command
}

type CageCommandsInput struct {
	Session *session.Session
	GlobalContext context.Context
}
type cageCommands struct {
	ses *session.Session
	globalContext context.Context
}

func NewCageCommands(input *CageCommandsInput) CageCommands {
	return &cageCommands{
		ses: input.Session,
		globalContext: input.GlobalContext,
	}
}
