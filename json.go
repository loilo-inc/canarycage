package main

// Optional deployment parameters that control how many tasks run during the
// deployment and the ordering of stopping and starting tasks.
type DeploymentConfiguration struct {
	_ struct{} `type:"structure"`

	// The upper limit (as a percentage of the service's desiredCount) of the number
	// of tasks that are allowed in the RUNNING or PENDING state in a service during
	// a deployment. The maximum number of tasks during a deployment is the desiredCount
	// multiplied by maximumPercent/100, rounded down to the nearest integer value.
	MaximumPercent int64 `locationName:"maximumPercent" type:"integer"`

	// The lower limit (as a percentage of the service's desiredCount) of the number
	// of running tasks that must remain in the RUNNING state in a service during
	// a deployment. The minimum number of healthy tasks during a deployment is
	// the desiredCount multiplied by minimumHealthyPercent/100, rounded up to the
	// nearest integer value.
	MinimumHealthyPercent int64 `locationName:"minimumHealthyPercent" type:"integer"`
}

type CreateServiceInput struct {
	_ struct{} `type:"structure"`

	// Unique, case-sensitive identifier that you provide to ensure the idempotency
	// of the request. Up to 32 ASCII characters are allowed.
	ClientToken string `locationName:"clientToken" type:"string"`

	// The short name or full Amazon Resource Name (ARN) of the cluster on which
	// to run your service. If you do not specify a cluster, the default cluster
	// is assumed.
	Cluster string `locationName:"cluster" type:"string"`

	// Optional deployment parameters that control how many tasks run during the
	// deployment and the ordering of stopping and starting tasks.
	DeploymentConfiguration DeploymentConfiguration `locationName:"deploymentConfiguration" type:"structure"`

	// The number of instantiations of the specified task definition to place and
	// keep running on your cluster.
	DesiredCount int64 `locationName:"desiredCount" type:"integer"`

	// The period of time, in seconds, that the Amazon ECS service scheduler should
	// ignore unhealthy Elastic Load Balancing target health checks after a task
	// has first started. This is only valid if your service is configured to use
	// a load balancer. If your service's tasks take a while to start and respond
	// to Elastic Load Balancing health checks, you can specify a health check grace
	// period of up to 7,200 seconds during which the ECS service scheduler ignores
	// health check status. This grace period can prevent the ECS service scheduler
	// from marking tasks as unhealthy and stopping them before they have time to
	// come up.
	HealthCheckGracePeriodSeconds int64 `locationName:"healthCheckGracePeriodSeconds" type:"integer"`

	// The launch type on which to run your service.
	LaunchType string `locationName:"launchType" type:"string" enum:"LaunchType"`

	// A load balancer object representing the load balancer to use with your service.
	// Currently, you are limited to one load balancer or target group per service.
	// After you create a service, the load balancer name or target group ARN, container
	// name, and container port specified in the service definition are immutable.
	//
	// For Classic Load Balancers, this object must contain the load balancer name,
	// the container name (as it appears in a container definition), and the container
	// port to access from the load balancer. When a task from this service is placed
	// on a container instance, the container instance is registered with the load
	// balancer specified here.
	//
	// For Application Load Balancers and Network Load Balancers, this object must
	// contain the load balancer target group ARN, the container name (as it appears
	// in a container definition), and the container port to access from the load
	// balancer. When a task from this service is placed on a container instance,
	// the container instance and port combination is registered as a target in
	// the target group specified here.
	//
	// Services with tasks that use the awsvpc network mode (for example, those
	// with the Fargate launch type) only support Application Load Balancers and
	// Network Load Balancers; Classic Load Balancers are not supported. Also, when
	// you create any target groups for these services, you must choose ip as the
	// target type, not instance, because tasks that use the awsvpc network mode
	// are associated with an elastic network interface, not an Amazon EC2 instance.
	LoadBalancers []LoadBalancer `locationName:"loadBalancers" type:"list"`

	// The network configuration for the service. This parameter is required for
	// task definitions that use the awsvpc network mode to receive their own Elastic
	// Network Interface, and it is not supported for other network modes. For more
	// information, see Task Networking (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking.html)
	// in the Amazon Elastic Container Service Developer Guide.
	NetworkConfiguration NetworkConfiguration `locationName:"networkConfiguration" type:"structure"`

	// An array of placement constraint objects to use for tasks in your service.
	// You can specify a maximum of 10 constraints per task (this limit includes
	// constraints in the task definition and those specified at run time).
	PlacementConstraints []PlacementConstraint `locationName:"placementConstraints" type:"list"`

	// The placement strategy objects to use for tasks in your service. You can
	// specify a maximum of five strategy rules per service.
	PlacementStrategy []PlacementStrategy `locationName:"placementStrategy" type:"list"`

	// The platform version on which to run your service. If one is not specified,
	// the latest version is used by default.
	PlatformVersion string `locationName:"platformVersion" type:"string"`

	// The name or full Amazon Resource Name (ARN) of the IAM role that allows Amazon
	// ECS to make calls to your load balancer on your behalf. This parameter is
	// only permitted if you are using a load balancer with your service and your
	// task definition does not use the awsvpc network mode. If you specify the
	// role parameter, you must also specify a load balancer object with the loadBalancers
	// parameter.
	//
	// If your account has already created the Amazon ECS service-linked role, that
	// role is used by default for your service unless you specify a role here.
	// The service-linked role is required if your task definition uses the awsvpc
	// network mode, in which case you should not specify a role here. For more
	// information, see Using Service-Linked Roles for Amazon ECS (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/using-service-linked-roles.html)
	// in the Amazon Elastic Container Service Developer Guide.
	//
	// If your specified role has a path other than /, then you must either specify
	// the full role ARN (this is recommended) or prefix the role name with the
	// path. For example, if a role with the name bar has a path of /foo/ then you
	// would specify /foo/bar as the role name. For more information, see Friendly
	// Names and Paths (http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html#identifiers-friendly-names)
	// in the IAM User Guide.
	Role string `locationName:"role" type:"string"`

	// The scheduling strategy to use for the service. For more information, see
	// Services (http://docs.aws.amazon.com/AmazonECS/latest/developerguideecs_services.html).
	//
	// There are two service scheduler strategies available:
	//
	//     REPLICA-The replica scheduling strategy places and maintains the desired
	//    number of tasks across your cluster. By default, the service scheduler
	//    spreads tasks across Availability Zones. You can use task placement strategies
	//    and constraints to customize task placement decisions.
	//
	//     DAEMON-The daemon scheduling strategy deploys exactly one task on each
	//    active container instance that meets all of the task placement constraints
	//    that you specify in your cluster. When using this strategy, there is no
	//    need to specify a desired number of tasks, a task placement strategy,
	//    or use Service Auto Scaling policies.
	//
	// Fargate tasks do not support the DAEMON scheduling strategy.
	SchedulingStrategy string `locationName:"schedulingStrategy" type:"string" enum:"SchedulingStrategy"`

	// The name of your service. Up to 255 letters (uppercase and lowercase), numbers,
	// hyphens, and underscores are allowed. Service names must be unique within
	// a cluster, but you can have similarly named services in multiple clusters
	// within a Region or across multiple Regions.
	//
	// ServiceName is a required field
	ServiceName string `locationName:"serviceName" type:"string" required:"true"`

	// The details of the service discovery registries to assign to this service.
	// For more information, see Service Discovery (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-discovery.html).
	//
	// Service discovery is supported for Fargate tasks if using platform version
	// v1.1.0 or later. For more information, see AWS Fargate Platform Versions
	// (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/platform_versions.html).
	ServiceRegistries []ServiceRegistry `locationName:"serviceRegistries" type:"list"`

	// The family and revision (family:revision) or full ARN of the task definition
	// to run in your service. If a revision is not specified, the latest ACTIVE
	// revision is used.
	//
	// TaskDefinition is a required field
	TaskDefinition string `locationName:"taskDefinition" type:"string" required:"true"`
}

type LoadBalancer struct {
	_ struct{} `type:"structure"`

	// The name of the container (as it appears in a container definition) to associate
	// with the load balancer.
	ContainerName string `locationName:"containerName" type:"string"`

	// The port on the container to associate with the load balancer. This port
	// must correspond to a containerPort in the service's task definition. Your
	// container instances must allow ingress traffic on the hostPort of the port
	// mapping.
	ContainerPort int64 `locationName:"containerPort" type:"integer"`

	// The name of a load balancer.
	LoadBalancerName string `locationName:"loadBalancerName" type:"string"`

	// The full Amazon Resource Name (ARN) of the Elastic Load Balancing target
	// group associated with a service.
	//
	// If your service's task definition uses the awsvpc network mode (which is
	// required for the Fargate launch type), you must choose ip as the target type,
	// not instance, because tasks that use the awsvpc network mode are associated
	// with an elastic network interface, not an Amazon EC2 instance.
	TargetGroupArn string `locationName:"targetGroupArn" type:"string"`
}

// Details of the service registry.
type ServiceRegistry struct {
	_ struct{} `type:"structure"`

	// The container name value, already specified in the task definition, to be
	// used for your service discovery service. If the task definition that your
	// service task specifies uses the bridge or host network mode, you must specify
	// a containerName and containerPort combination from the task definition. If
	// the task definition that your service task specifies uses the awsvpc network
	// mode and a type SRV DNS record is used, you must specify either a containerName
	// and containerPort combination or a port value, but not both.
	ContainerName string `locationName:"containerName" type:"string"`

	// The port value, already specified in the task definition, to be used for
	// your service discovery service. If the task definition your service task
	// specifies uses the bridge or host network mode, you must specify a containerName
	// and containerPort combination from the task definition. If the task definition
	// your service task specifies uses the awsvpc network mode and a type SRV DNS
	// record is used, you must specify either a containerName and containerPort
	// combination or a port value, but not both.
	ContainerPort int64 `locationName:"containerPort" type:"integer"`

	// The port value used if your service discovery service specified an SRV record.
	// This field is required if both the awsvpc network mode and SRV records are
	// used.
	Port int64 `locationName:"port" type:"integer"`

	// The Amazon Resource Name (ARN) of the service registry. The currently supported
	// service registry is Amazon Route 53 Auto Naming. For more information, see
	// Service (https://docs.aws.amazon.com/Route53/latest/APIReference/API_autonaming_Service.html).
	RegistryArn string `locationName:"registryArn" type:"string"`
}
// An object representing the network configuration for a task or service.
type NetworkConfiguration struct {
	_ struct{} `type:"structure"`

	// The VPC subnets and security groups associated with a task.
	//
	// All specified subnets and security groups must be from the same VPC.
	AwsvpcConfiguration AwsVpcConfiguration `locationName:"awsvpcConfiguration" type:"structure"`
}
// An object representing the networking details for a task or service.
type AwsVpcConfiguration struct {
	_ struct{} `type:"structure"`

	// Whether the task's elastic network interface receives a public IP address.
	AssignPublicIp string `locationName:"assignPublicIp" type:"string" enum:"AssignPublicIp"`

	// The security groups associated with the task or service. If you do not specify
	// a security group, the default security group for the VPC is used. There is
	// a limit of 5 security groups able to be specified per AwsVpcConfiguration.
	//
	// All specified security groups must be from the same VPC.
	SecurityGroups []string `locationName:"securityGroups" type:"list"`

	// The subnets associated with the task or service. There is a limit of 10 subnets
	// able to be specified per AwsVpcConfiguration.
	//
	// All specified subnets must be from the same VPC.
	//
	// Subnets is a required field
	Subnets []string `locationName:"subnets" type:"list" required:"true"`
}

type PlacementStrategy struct {
	_ struct{} `type:"structure"`

	// The field to apply the placement strategy against. For the spread placement
	// strategy, valid values are instanceId (or host, which has the same effect),
	// or any platform or custom attribute that is applied to a container instance,
	// such as attribute:ecs.availability-zone. For the binpack placement strategy,
	// valid values are cpu and memory. For the random placement strategy, this
	// field is not used.
	Field string `locationName:"field" type:"string"`

	// The type of placement strategy. The random placement strategy randomly places
	// tasks on available candidates. The spread placement strategy spreads placement
	// across available candidates evenly based on the field parameter. The binpack
	// strategy places tasks on available candidates that have the least available
	// amount of the resource that is specified with the field parameter. For example,
	// if you binpack on memory, a task is placed on the instance with the least
	// amount of remaining memory (but still enough to run the task).
	Type string `locationName:"type" type:"string" enum:"PlacementStrategyType"`
}
type PlacementConstraint struct {
	_ struct{} `type:"structure"`

	// A cluster query language expression to apply to the constraint. You cannot
	// specify an expression if the constraint type is distinctInstance. For more
	// information, see Cluster Query Language (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/cluster-query-language.html)
	// in the Amazon Elastic Container Service Developer Guide.
	Expression string `locationName:"expression" type:"string"`

	// The type of constraint. Use distinctInstance to ensure that each task in
	// a particular group is running on a different container instance. Use memberOf
	// to restrict the selection to a group of valid candidates. The value distinctInstance
	// is not supported in task definitions.
	Type string `locationName:"type" type:"string" enum:"PlacementConstraintType"`
}

// Register Task Definition

type RegisterTaskDefinitionInput struct {
	_ struct{} `type:"structure"`

	// A list of container definitions in JSON format that describe the different
	// containers that make up your task.
	//
	// ContainerDefinitions is a required field
	ContainerDefinitions []*ContainerDefinition `locationName:"containerDefinitions" type:"list" required:"true"`

	// The number of CPU units used by the task. It can be expressed as an integer
	// using CPU units, for example 1024, or as a string using vCPUs, for example
	// 1 vCPU or 1 vcpu, in a task definition. String values are converted to an
	// integer indicating the CPU units when the task definition is registered.
	//
	// Task-level CPU and memory parameters are ignored for Windows containers.
	// We recommend specifying container-level resources for Windows containers.
	//
	// If using the EC2 launch type, this field is optional. Supported values are
	// between 128 CPU units (0.125 vCPUs) and 10240 CPU units (10 vCPUs).
	//
	// If using the Fargate launch type, this field is required and you must use
	// one of the following values, which determines your range of supported values
	// for the memory parameter:
	//
	//    * 256 (.25 vCPU) - Available memory values: 512 (0.5 GB), 1024 (1 GB),
	//    2048 (2 GB)
	//
	//    * 512 (.5 vCPU) - Available memory values: 1024 (1 GB), 2048 (2 GB), 3072
	//    (3 GB), 4096 (4 GB)
	//
	//    * 1024 (1 vCPU) - Available memory values: 2048 (2 GB), 3072 (3 GB), 4096
	//    (4 GB), 5120 (5 GB), 6144 (6 GB), 7168 (7 GB), 8192 (8 GB)
	//
	//    * 2048 (2 vCPU) - Available memory values: Between 4096 (4 GB) and 16384
	//    (16 GB) in increments of 1024 (1 GB)
	//
	//    * 4096 (4 vCPU) - Available memory values: Between 8192 (8 GB) and 30720
	//    (30 GB) in increments of 1024 (1 GB)
	Cpu *string `locationName:"cpu" type:"string"`

	// The Amazon Resource Name (ARN) of the task execution role that the Amazon
	// ECS container agent and the Docker daemon can assume.
	ExecutionRoleArn *string `locationName:"executionRoleArn" type:"string"`

	// You must specify a family for a task definition, which allows you to track
	// multiple versions of the same task definition. The family is used as a name
	// for your task definition. Up to 255 letters (uppercase and lowercase), numbers,
	// hyphens, and underscores are allowed.
	//
	// Family is a required field
	Family *string `locationName:"family" type:"string" required:"true"`

	// The amount of memory (in MiB) used by the task. It can be expressed as an
	// integer using MiB, for example 1024, or as a string using GB, for example
	// 1GB or 1 GB, in a task definition. String values are converted to an integer
	// indicating the MiB when the task definition is registered.
	//
	// Task-level CPU and memory parameters are ignored for Windows containers.
	// We recommend specifying container-level resources for Windows containers.
	//
	// If using the EC2 launch type, this field is optional.
	//
	// If using the Fargate launch type, this field is required and you must use
	// one of the following values, which determines your range of supported values
	// for the cpu parameter:
	//
	//    * 512 (0.5 GB), 1024 (1 GB), 2048 (2 GB) - Available cpu values: 256 (.25
	//    vCPU)
	//
	//    * 1024 (1 GB), 2048 (2 GB), 3072 (3 GB), 4096 (4 GB) - Available cpu values:
	//    512 (.5 vCPU)
	//
	//    * 2048 (2 GB), 3072 (3 GB), 4096 (4 GB), 5120 (5 GB), 6144 (6 GB), 7168
	//    (7 GB), 8192 (8 GB) - Available cpu values: 1024 (1 vCPU)
	//
	//    * Between 4096 (4 GB) and 16384 (16 GB) in increments of 1024 (1 GB) -
	//    Available cpu values: 2048 (2 vCPU)
	//
	//    * Between 8192 (8 GB) and 30720 (30 GB) in increments of 1024 (1 GB) -
	//    Available cpu values: 4096 (4 vCPU)
	Memory *string `locationName:"memory" type:"string"`

	// The Docker networking mode to use for the containers in the task. The valid
	// values are none, bridge, awsvpc, and host. The default Docker network mode
	// is bridge. If using the Fargate launch type, the awsvpc network mode is required.
	// If using the EC2 launch type, any network mode can be used. If the network
	// mode is set to none, you can't specify port mappings in your container definitions,
	// and the task's containers do not have external connectivity. The host and
	// awsvpc network modes offer the highest networking performance for containers
	// because they use the EC2 network stack instead of the virtualized network
	// stack provided by the bridge mode.
	//
	// With the host and awsvpc network modes, exposed container ports are mapped
	// directly to the corresponding host port (for the host network mode) or the
	// attached elastic network interface port (for the awsvpc network mode), so
	// you cannot take advantage of dynamic host port mappings.
	//
	// If the network mode is awsvpc, the task is allocated an Elastic Network Interface,
	// and you must specify a NetworkConfiguration when you create a service or
	// run a task with the task definition. For more information, see Task Networking
	// (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking.html)
	// in the Amazon Elastic Container Service Developer Guide.
	//
	// If the network mode is host, you can't run multiple instantiations of the
	// same task on a single container instance when port mappings are used.
	//
	// Docker for Windows uses different network modes than Docker for Linux. When
	// you register a task definition with Windows containers, you must not specify
	// a network mode.
	//
	// For more information, see Network settings (https://docs.docker.com/engine/reference/run/#network-settings)
	// in the Docker run reference.
	NetworkMode *string `locationName:"networkMode" type:"string" enum:"NetworkMode"`

	// An array of placement constraint objects to use for the task. You can specify
	// a maximum of 10 constraints per task (this limit includes constraints in
	// the task definition and those specified at run time).
	PlacementConstraints []*TaskDefinitionPlacementConstraint `locationName:"placementConstraints" type:"list"`

	// The launch type required by the task. If no value is specified, it defaults
	// to EC2.
	RequiresCompatibilities []*string `locationName:"requiresCompatibilities" type:"list"`

	// The short name or full Amazon Resource Name (ARN) of the IAM role that containers
	// in this task can assume. All containers in this task are granted the permissions
	// that are specified in this role. For more information, see IAM Roles for
	// Tasks (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html)
	// in the Amazon Elastic Container Service Developer Guide.
	TaskRoleArn *string `locationName:"taskRoleArn" type:"string"`

	// A list of volume definitions in JSON format that containers in your task
	// may use.
	Volumes []*Volume `locationName:"volumes" type:"list"`
}

type ContainerDefinition struct {
	_ struct{} `type:"structure"`

	// The command that is passed to the container. This parameter maps to Cmd in
	// the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the COMMAND parameter to docker run (https://docs.docker.com/engine/reference/run/).
	// For more information, see https://docs.docker.com/engine/reference/builder/#cmd
	// (https://docs.docker.com/engine/reference/builder/#cmd).
	Command []*string `locationName:"command" type:"list"`

	// The number of cpu units reserved for the container. This parameter maps to
	// CpuShares in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --cpu-shares option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// This field is optional for tasks using the Fargate launch type, and the only
	// requirement is that the total amount of CPU reserved for all containers within
	// a task be lower than the task-level cpu value.
	//
	// You can determine the number of CPU units that are available per EC2 instance
	// type by multiplying the vCPUs listed for that instance type on the Amazon
	// EC2 Instances (http://aws.amazon.com/ec2/instance-types/) detail page by
	// 1,024.
	//
	// For example, if you run a single-container task on a single-core instance
	// type with 512 CPU units specified for that container, and that is the only
	// task running on the container instance, that container could use the full
	// 1,024 CPU unit share at any given time. However, if you launched another
	// copy of the same task on that container instance, each task would be guaranteed
	// a minimum of 512 CPU units when needed, and each container could float to
	// higher CPU usage if the other container was not using it, but if both tasks
	// were 100% active all of the time, they would be limited to 512 CPU units.
	//
	// Linux containers share unallocated CPU units with other containers on the
	// container instance with the same ratio as their allocated amount. For example,
	// if you run a single-container task on a single-core instance type with 512
	// CPU units specified for that container, and that is the only task running
	// on the container instance, that container could use the full 1,024 CPU unit
	// share at any given time. However, if you launched another copy of the same
	// task on that container instance, each task would be guaranteed a minimum
	// of 512 CPU units when needed, and each container could float to higher CPU
	// usage if the other container was not using it, but if both tasks were 100%
	// active all of the time, they would be limited to 512 CPU units.
	//
	// On Linux container instances, the Docker daemon on the container instance
	// uses the CPU value to calculate the relative CPU share ratios for running
	// containers. For more information, see CPU share constraint (https://docs.docker.com/engine/reference/run/#cpu-share-constraint)
	// in the Docker documentation. The minimum valid CPU share value that the Linux
	// kernel allows is 2; however, the CPU parameter is not required, and you can
	// use CPU values below 2 in your container definitions. For CPU values below
	// 2 (including null), the behavior varies based on your Amazon ECS container
	// agent version:
	//
	//    * Agent versions less than or equal to 1.1.0: Null and zero CPU values
	//    are passed to Docker as 0, which Docker then converts to 1,024 CPU shares.
	//    CPU values of 1 are passed to Docker as 1, which the Linux kernel converts
	//    to 2 CPU shares.
	//
	//    * Agent versions greater than or equal to 1.2.0: Null, zero, and CPU values
	//    of 1 are passed to Docker as 2.
	//
	// On Windows container instances, the CPU limit is enforced as an absolute
	// limit, or a quota. Windows containers only have access to the specified amount
	// of CPU that is described in the task definition.
	Cpu *int64 `locationName:"cpu" type:"integer"`

	// When this parameter is true, networking is disabled within the container.
	// This parameter maps to NetworkDisabled in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/).
	//
	// This parameter is not supported for Windows containers.
	DisableNetworking *bool `locationName:"disableNetworking" type:"boolean"`

	// A list of DNS search domains that are presented to the container. This parameter
	// maps to DnsSearch in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --dns-search option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// This parameter is not supported for Windows containers.
	DnsSearchDomains []*string `locationName:"dnsSearchDomains" type:"list"`

	// A list of DNS servers that are presented to the container. This parameter
	// maps to Dns in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --dns option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// This parameter is not supported for Windows containers.
	DnsServers []*string `locationName:"dnsServers" type:"list"`

	// A key/value map of labels to add to the container. This parameter maps to
	// Labels in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --label option to docker run (https://docs.docker.com/engine/reference/run/).
	// This parameter requires version 1.18 of the Docker Remote API or greater
	// on your container instance. To check the Docker Remote API version on your
	// container instance, log in to your container instance and run the following
	// command: sudo docker version | grep "Server API version"
	DockerLabels map[string]*string `locationName:"dockerLabels" type:"map"`

	// A list of strings to provide custom labels for SELinux and AppArmor multi-level
	// security systems. This field is not valid for containers in tasks using the
	// Fargate launch type.
	//
	// This parameter maps to SecurityOpt in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --security-opt option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// The Amazon ECS container agent running on a container instance must register
	// with the ECS_SELINUX_CAPABLE=true or ECS_APPARMOR_CAPABLE=true environment
	// variables before containers placed on that instance can use these security
	// options. For more information, see Amazon ECS Container Agent Configuration
	// (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-agent-config.html)
	// in the Amazon Elastic Container Service Developer Guide.
	//
	// This parameter is not supported for Windows containers.
	DockerSecurityOptions []*string `locationName:"dockerSecurityOptions" type:"list"`

	// Early versions of the Amazon ECS container agent do not properly handle entryPoint
	// parameters. If you have problems using entryPoint, update your container
	// agent or enter your commands and arguments as command array items instead.
	//
	// The entry point that is passed to the container. This parameter maps to Entrypoint
	// in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --entrypoint option to docker run (https://docs.docker.com/engine/reference/run/).
	// For more information, see https://docs.docker.com/engine/reference/builder/#entrypoint
	// (https://docs.docker.com/engine/reference/builder/#entrypoint).
	EntryPoint []*string `locationName:"entryPoint" type:"list"`

	// The environment variables to pass to a container. This parameter maps to
	// Env in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --env option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// We do not recommend using plaintext environment variables for sensitive information,
	// such as credential data.
	Environment []*KeyValuePair `locationName:"environment" type:"list"`

	// If the essential parameter of a container is marked as true, and that container
	// fails or stops for any reason, all other containers that are part of the
	// task are stopped. If the essential parameter of a container is marked as
	// false, then its failure does not affect the rest of the containers in a task.
	// If this parameter is omitted, a container is assumed to be essential.
	//
	// All tasks must have at least one essential container. If you have an application
	// that is composed of multiple containers, you should group containers that
	// are used for a common purpose into components, and separate the different
	// components into multiple task definitions. For more information, see Application
	// Architecture (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/application_architecture.html)
	// in the Amazon Elastic Container Service Developer Guide.
	Essential *bool `locationName:"essential" type:"boolean"`

	// A list of hostnames and IP address mappings to append to the /etc/hosts file
	// on the container. If using the Fargate launch type, this may be used to list
	// non-Fargate hosts to which the container can talk. This parameter maps to
	// ExtraHosts in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --add-host option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// This parameter is not supported for Windows containers.
	ExtraHosts []*HostEntry `locationName:"extraHosts" type:"list"`

	// The health check command and associated configuration parameters for the
	// container. This parameter maps to HealthCheck in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the HEALTHCHECK parameter of docker run (https://docs.docker.com/engine/reference/run/).
	HealthCheck *HealthCheck `locationName:"healthCheck" type:"structure"`

	// The hostname to use for your container. This parameter maps to Hostname in
	// the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --hostname option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// The hostname parameter is not supported if using the awsvpc networkMode.
	Hostname *string `locationName:"hostname" type:"string"`

	// The image used to start a container. This string is passed directly to the
	// Docker daemon. Images in the Docker Hub registry are available by default.
	// Other repositories are specified with either repository-url/image:tag or
	// repository-url/image@digest. Up to 255 letters (uppercase and lowercase),
	// numbers, hyphens, underscores, colons, periods, forward slashes, and number
	// signs are allowed. This parameter maps to Image in the Create a container
	// (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the IMAGE parameter of docker run (https://docs.docker.com/engine/reference/run/).
	//
	//    * When a new task starts, the Amazon ECS container agent pulls the latest
	//    version of the specified image and tag for the container to use. However,
	//    subsequent updates to a repository image are not propagated to already
	//    running tasks.
	//
	//    * Images in Amazon ECR repositories can be specified by either using the
	//    full registry/repository:tag or registry/repository@digest. For example,
	//    012345678910.dkr.ecr.<region-name>.amazonaws.com/<repository-name>:latest
	//    or 012345678910.dkr.ecr.<region-name>.amazonaws.com/<repository-name>@sha256:94afd1f2e64d908bc90dbca0035a5b567EXAMPLE.
	//
	//
	//    * Images in official repositories on Docker Hub use a single name (for
	//    example, ubuntu or mongo).
	//
	//    * Images in other repositories on Docker Hub are qualified with an organization
	//    name (for example, amazon/amazon-ecs-agent).
	//
	//    * Images in other online repositories are qualified further by a domain
	//    name (for example, quay.io/assemblyline/ubuntu).
	Image *string `locationName:"image" type:"string"`

	// The link parameter allows containers to communicate with each other without
	// the need for port mappings. Only supported if the network mode of a task
	// definition is set to bridge. The name:internalName construct is analogous
	// to name:alias in Docker links. Up to 255 letters (uppercase and lowercase),
	// numbers, hyphens, and underscores are allowed. For more information about
	// linking Docker containers, go to https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/
	// (https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/).
	// This parameter maps to Links in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --link option to docker run (https://docs.docker.com/engine/reference/commandline/run/).
	//
	// This parameter is not supported for Windows containers.
	//
	// Containers that are collocated on a single container instance may be able
	// to communicate with each other without requiring links or host port mappings.
	// Network isolation is achieved on the container instance using security groups
	// and VPC settings.
	Links []*string `locationName:"links" type:"list"`

	// Linux-specific modifications that are applied to the container, such as Linux
	// KernelCapabilities.
	//
	// This parameter is not supported for Windows containers.
	LinuxParameters *LinuxParameters `locationName:"linuxParameters" type:"structure"`

	// The log configuration specification for the container.
	//
	// If using the Fargate launch type, the only supported value is awslogs.
	//
	// This parameter maps to LogConfig in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --log-driver option to docker run (https://docs.docker.com/engine/reference/run/).
	// By default, containers use the same logging driver that the Docker daemon
	// uses; however the container may use a different logging driver than the Docker
	// daemon by specifying a log driver with this parameter in the container definition.
	// To use a different logging driver for a container, the log system must be
	// configured properly on the container instance (or on a different log server
	// for remote logging options). For more information on the options for different
	// supported log drivers, see Configure logging drivers (https://docs.docker.com/engine/admin/logging/overview/)
	// in the Docker documentation.
	//
	// Amazon ECS currently supports a subset of the logging drivers available to
	// the Docker daemon (shown in the LogConfiguration data type). Additional log
	// drivers may be available in future releases of the Amazon ECS container agent.
	//
	// This parameter requires version 1.18 of the Docker Remote API or greater
	// on your container instance. To check the Docker Remote API version on your
	// container instance, log in to your container instance and run the following
	// command: sudo docker version | grep "Server API version"
	//
	// The Amazon ECS container agent running on a container instance must register
	// the logging drivers available on that instance with the ECS_AVAILABLE_LOGGING_DRIVERS
	// environment variable before containers placed on that instance can use these
	// log configuration options. For more information, see Amazon ECS Container
	// Agent Configuration (http://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-agent-config.html)
	// in the Amazon Elastic Container Service Developer Guide.
	LogConfiguration *LogConfiguration `locationName:"logConfiguration" type:"structure"`

	// The hard limit (in MiB) of memory to present to the container. If your container
	// attempts to exceed the memory specified here, the container is killed. This
	// parameter maps to Memory in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --memory option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// If your containers are part of a task using the Fargate launch type, this
	// field is optional and the only requirement is that the total amount of memory
	// reserved for all containers within a task be lower than the task memory value.
	//
	// For containers that are part of a task using the EC2 launch type, you must
	// specify a non-zero integer for one or both of memory or memoryReservation
	// in container definitions. If you specify both, memory must be greater than
	// memoryReservation. If you specify memoryReservation, then that value is subtracted
	// from the available memory resources for the container instance on which the
	// container is placed; otherwise, the value of memory is used.
	//
	// The Docker daemon reserves a minimum of 4 MiB of memory for a container,
	// so you should not specify fewer than 4 MiB of memory for your containers.
	Memory *int64 `locationName:"memory" type:"integer"`

	// The soft limit (in MiB) of memory to reserve for the container. When system
	// memory is under heavy contention, Docker attempts to keep the container memory
	// to this soft limit; however, your container can consume more memory when
	// it needs to, up to either the hard limit specified with the memory parameter
	// (if applicable), or all of the available memory on the container instance,
	// whichever comes first. This parameter maps to MemoryReservation in the Create
	// a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --memory-reservation option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// You must specify a non-zero integer for one or both of memory or memoryReservation
	// in container definitions. If you specify both, memory must be greater than
	// memoryReservation. If you specify memoryReservation, then that value is subtracted
	// from the available memory resources for the container instance on which the
	// container is placed; otherwise, the value of memory is used.
	//
	// For example, if your container normally uses 128 MiB of memory, but occasionally
	// bursts to 256 MiB of memory for short periods of time, you can set a memoryReservation
	// of 128 MiB, and a memory hard limit of 300 MiB. This configuration would
	// allow the container to only reserve 128 MiB of memory from the remaining
	// resources on the container instance, but also allow the container to consume
	// more memory resources when needed.
	//
	// The Docker daemon reserves a minimum of 4 MiB of memory for a container,
	// so you should not specify fewer than 4 MiB of memory for your containers.
	MemoryReservation *int64 `locationName:"memoryReservation" type:"integer"`

	// The mount points for data volumes in your container.
	//
	// This parameter maps to Volumes in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --volume option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// Windows containers can mount whole directories on the same drive as $env:ProgramData.
	// Windows containers cannot mount directories on a different drive, and mount
	// point cannot be across drives.
	MountPoints []*MountPoint `locationName:"mountPoints" type:"list"`

	// The name of a container. If you are linking multiple containers together
	// in a task definition, the name of one container can be entered in the links
	// of another container to connect the containers. Up to 255 letters (uppercase
	// and lowercase), numbers, hyphens, and underscores are allowed. This parameter
	// maps to name in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --name option to docker run (https://docs.docker.com/engine/reference/run/).
	Name *string `locationName:"name" type:"string"`

	// The list of port mappings for the container. Port mappings allow containers
	// to access ports on the host container instance to send or receive traffic.
	//
	// For task definitions that use the awsvpc network mode, you should only specify
	// the containerPort. The hostPort can be left blank or it must be the same
	// value as the containerPort.
	//
	// Port mappings on Windows use the NetNAT gateway address rather than localhost.
	// There is no loopback for port mappings on Windows, so you cannot access a
	// container's mapped port from the host itself.
	//
	// This parameter maps to PortBindings in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --publish option to docker run (https://docs.docker.com/engine/reference/run/).
	// If the network mode of a task definition is set to none, then you can't specify
	// port mappings. If the network mode of a task definition is set to host, then
	// host ports must either be undefined or they must match the container port
	// in the port mapping.
	//
	// After a task reaches the RUNNING status, manual and automatic host and container
	// port assignments are visible in the Network Bindings section of a container
	// description for a selected task in the Amazon ECS console. The assignments
	// are also visible in the networkBindings section DescribeTasks responses.
	PortMappings []*PortMapping `locationName:"portMappings" type:"list"`

	// When this parameter is true, the container is given elevated privileges on
	// the host container instance (similar to the root user). This parameter maps
	// to Privileged in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --privileged option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// This parameter is not supported for Windows containers or tasks using the
	// Fargate launch type.
	Privileged *bool `locationName:"privileged" type:"boolean"`

	// When this parameter is true, the container is given read-only access to its
	// root file system. This parameter maps to ReadonlyRootfs in the Create a container
	// (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --read-only option to docker run.
	//
	// This parameter is not supported for Windows containers.
	ReadonlyRootFilesystem *bool `locationName:"readonlyRootFilesystem" type:"boolean"`

	// The private repository authentication credentials to use.
	RepositoryCredentials *RepositoryCredentials `locationName:"repositoryCredentials" type:"structure"`

	// A list of ulimits to set in the container. This parameter maps to Ulimits
	// in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --ulimit option to docker run (https://docs.docker.com/engine/reference/run/).
	// Valid naming values are displayed in the Ulimit data type. This parameter
	// requires version 1.18 of the Docker Remote API or greater on your container
	// instance. To check the Docker Remote API version on your container instance,
	// log in to your container instance and run the following command: sudo docker
	// version | grep "Server API version"
	//
	// This parameter is not supported for Windows containers.
	Ulimits []*Ulimit `locationName:"ulimits" type:"list"`

	// The user name to use inside the container. This parameter maps to User in
	// the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --user option to docker run (https://docs.docker.com/engine/reference/run/).
	//
	// This parameter is not supported for Windows containers.
	User *string `locationName:"user" type:"string"`

	// Data volumes to mount from another container. This parameter maps to VolumesFrom
	// in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --volumes-from option to docker run (https://docs.docker.com/engine/reference/run/).
	//VolumesFrom []*VolumeFrom `locationName:"volumesFrom" type:"list"`

	// The working directory in which to run commands inside the container. This
	// parameter maps to WorkingDir in the Create a container (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/#create-a-container)
	// section of the Docker Remote API (https://docs.docker.com/engine/reference/api/docker_remote_api_v1.27/)
	// and the --workdir option to docker run (https://docs.docker.com/engine/reference/run/).
	WorkingDirectory *string `locationName:"workingDirectory" type:"string"`
}
// A data volume used in a task definition.
type Volume struct {
	_ struct{} `type:"structure"`

	// The contents of the host parameter determine whether your data volume persists
	// on the host container instance and where it is stored. If the host parameter
	// is empty, then the Docker daemon assigns a host path for your data volume,
	// but the data is not guaranteed to persist after the containers associated
	// with it stop running.
	//
	// Windows containers can mount whole directories on the same drive as $env:ProgramData.
	// Windows containers cannot mount directories on a different drive, and mount
	// point cannot be across drives. For example, you can mount C:\my\path:C:\my\path
	// and D:\:D:\, but not D:\my\path:C:\my\path or D:\:C:\my\path.
	Host *HostVolumeProperties `locationName:"host" type:"structure"`

	// The name of the volume. Up to 255 letters (uppercase and lowercase), numbers,
	// hyphens, and underscores are allowed. This name is referenced in the sourceVolume
	// parameter of container definition mountPoints.
	Name *string `locationName:"name" type:"string"`
}

// The ulimit settings to pass to the container.
type Ulimit struct {
	_ struct{} `type:"structure"`

	// The hard limit for the ulimit type.
	//
	// HardLimit is a required field
	HardLimit *int64 `locationName:"hardLimit" type:"integer" required:"true"`

	// The type of the ulimit.
	//
	// Name is a required field
	Name *string `locationName:"name" type:"string" required:"true" enum:"UlimitName"`

	// The soft limit for the ulimit type.
	//
	// SoftLimit is a required field
	SoftLimit *int64 `locationName:"softLimit" type:"integer" required:"true"`
}


// Details on a container instance host volume.
type HostVolumeProperties struct {
	_ struct{} `type:"structure"`

	// The path on the host container instance that is presented to the container.
	// If this parameter is empty, then the Docker daemon has assigned a host path
	// for you. If the host parameter contains a sourcePath file location, then
	// the data volume persists at the specified location on the host container
	// instance until you delete it manually. If the sourcePath value does not exist
	// on the host container instance, the Docker daemon creates it. If the location
	// does exist, the contents of the source path folder are exported.
	//
	// If you are using the Fargate launch type, the sourcePath parameter is not
	// supported.
	SourcePath *string `locationName:"sourcePath" type:"string"`
}
type PortMapping struct {
	_ struct{} `type:"structure"`

	// The port number on the container that is bound to the user-specified or automatically
	// assigned host port.
	//
	// If using containers in a task with the awsvpc or host network mode, exposed
	// ports should be specified using containerPort.
	//
	// If using containers in a task with the bridge network mode and you specify
	// a container port and not a host port, your container automatically receives
	// a host port in the ephemeral port range (for more information, see hostPort).
	// Port mappings that are automatically assigned in this way do not count toward
	// the 100 reserved ports limit of a container instance.
	ContainerPort *int64 `locationName:"containerPort" type:"integer"`

	// The port number on the container instance to reserve for your container.
	//
	// If using containers in a task with the awsvpc or host network mode, the hostPort
	// can either be left blank or set to the same value as the containerPort.
	//
	// If using containers in a task with the bridge network mode, you can specify
	// a non-reserved host port for your container port mapping, or you can omit
	// the hostPort (or set it to 0) while specifying a containerPort and your container
	// automatically receives a port in the ephemeral port range for your container
	// instance operating system and Docker version.
	//
	// The default ephemeral port range for Docker version 1.6.0 and later is listed
	// on the instance under /proc/sys/net/ipv4/ip_local_port_range; if this kernel
	// parameter is unavailable, the default ephemeral port range from 49153 through
	// 65535 is used. You should not attempt to specify a host port in the ephemeral
	// port range as these are reserved for automatic assignment. In general, ports
	// below 32768 are outside of the ephemeral port range.
	//
	// The default ephemeral port range from 49153 through 65535 is always used
	// for Docker versions before 1.6.0.
	//
	// The default reserved ports are 22 for SSH, the Docker ports 2375 and 2376,
	// and the Amazon ECS container agent ports 51678 and 51679. Any host port that
	// was previously specified in a running task is also reserved while the task
	// is running (after a task stops, the host port is released). The current reserved
	// ports are displayed in the remainingResources of DescribeContainerInstances
	// output, and a container instance may have up to 100 reserved ports at a time,
	// including the default reserved ports (automatically assigned ports do not
	// count toward the 100 reserved ports limit).
	HostPort *int64 `locationName:"hostPort" type:"integer"`

	// The protocol used for the port mapping. Valid values are tcp and udp. The
	// default is tcp.
	Protocol *string `locationName:"protocol" type:"string" enum:"TransportProtocol"`
}
