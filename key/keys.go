package key

type DepsKey string

const (
	EcsCli      DepsKey = "ecs"
	EcrCli      DepsKey = "ecr"
	Ec2Cli      DepsKey = "ec2"
	AlbCli      DepsKey = "alb"
	Logger      DepsKey = "logger"
	Scanner     DepsKey = "scanner"
	Env         DepsKey = "env"
	Time        DepsKey = "time"
	TaskFactory DepsKey = "task-factory"
)
