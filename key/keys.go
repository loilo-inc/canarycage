package key

type DepsKey string

const (
	EcsCli      DepsKey = "ecs"
	Ec2Cli      DepsKey = "ec2"
	AlbCli      DepsKey = "alb"
	Env         DepsKey = "env"
	Time        DepsKey = "time"
	TaskFactory DepsKey = "task-factory"
)
