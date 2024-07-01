package key

type DepsKey string

const (
	EcsCli         DepsKey = "ecs"
	Ec2Cli         DepsKey = "ec2"
	SrvCli         DepsKey = "srv"
	AlbCli         DepsKey = "alb"
	Env            DepsKey = "env"
	TimeoutManager DepsKey = "timeout-manager"
	Time           DepsKey = "time"
	TaskFactory    DepsKey = "task-factory"
)
