# canarycage

![](https://github.com/loilo-inc/canarycage/workflows/CI/badge.svg)
[![codecov](https://codecov.io/gh/loilo-inc/canarycage/branch/main/graph/badge.svg?token=WRW1qemxSR)](https://codecov.io/gh/loilo-inc/canarycage)
![https://img.shields.io/github/tag/loilo-inc/canarycage.svg](https://img.shields.io/github/tag/loilo-inc/canarycage.svg)
[![license](https://img.shields.io/github/license/loilo-inc/canarycage.svg)](https://github.com/loilo-inc/canarycage)

A deployment tool for AWS ECS

## Description

`canarycage` (aka. `cage`) is a deployment tool for AWS ECS. It does the canary deployment of the task to the service before updating the service. This tool is designed to be a robust and reliable deployment tool for ECS.

## Install

via `go install` (Recommended)

```bash
$ go install github.com/loilo-inc/canarycage/cli/cage@latest
$ cage upgrade
```

dowload from Github Releases

```
$ curl -oL https://github.com/loilo-inc/canarycage/releases/download/${VERSION}/canarycage_{linux|darwin}_{amd64|arm64}.zip
$ unzip canarycage_linux_amd64.zip
$ chmod +x cage
$ mv cage /usr/local/bin/cage
```

## Usage

canarycage needs two JSON files to deploy the canary task, `service.json` and `task-definition.json`.
Both files are structured as same as `aws ecs create-service` and `aws ecs register-task-definition` command's input. Those files are required to be placed in the same directory as below:

```txt
deploy/
  |- service.json
  |- task-definition.json
```

We recommend managing those files in the same repository as the source code of the service for continuous deployment.

### cage up

`up` command will create a new service with a new task definition. This command is useful for the first deployment. Basic usage is as follows:

```
$ cage up --region ${AWS_REGION} ./deploy
```

the first argument is the path to the directory that contains `service.json` and `task-definition.json`.
`--region` flag is required for all commands as well as `AWS_REGION` environment variable.

### cage rollout

`rollout` command will update existing service with a new task definition. This command is similar to `aws ecs update-service` command but it has some "additional" features for safe deployment. Basic usage is as follows:

#### Fargate

For Fargate, you can execute the command as below:

```bash
$ cage rollout --region ${AWS_REGION} ./deploy
```

#### EC2

For EC2, you need to specify `--canaryInstanceArn` flag to specify the instance that will run canary task.

```bash
$ cage rollout --region ${AWS_REGION} --canaryInstanceArn i-abcdef123456
```

During the deployment, `canarycage` will launch a canary task with the same network configuration as the existing service. If the canary task is healthy, the service will be updated with the new task definition. If the canary task is unhealthy, the service will remain in the previous state.

Evaluation of the canary task depends on service and task definition. Currently `canarycage` supports the following evaluation:

#### Custom health check

If any container in the task definition has a health check, `canarycage` will evaluate each container's health check. If all health checks are passed, the canary task will be considered healthy.

#### ALB Target Group helth check

If the service has an Application Load Balancer, `canarycage` will register the canary task to the target group of the service's load balancer. The canary task will be evaluated by the health check of the target group and deregistered either when the task's state becomes `HEALTHY` or `UNHEALTHY`.

If the service is not attached to any target group, this evaluation will be skipped. Instead, `canarycage` will wait for a while (value from `--canaryTaskIdleDuration` flag) and advance to the next step.

#### `--updateService` flag

By default, `cage rollout` will only update the task definition of the service. If you want to update the service as well, you can specify `--updateService` flag. This flag will update the service with the service definition in the `service.json` file. This is useful when you want to update the service's network configuration, load balancer configuration, or other service-level configurations.

### cage audit

`audit` command scans container images used by an ECS service and summarizes ECR image scan findings. Basic usage is as follows:

```bash
$ cage audit --region ${AWS_REGION} ./deploy
```

You can also specify the target service directly:

```bash
$ cage audit --region ${AWS_REGION} --cluster ${ECS_CLUSTER} --service ${ECS_SERVICE}
```

By default, the output is a table with per-container status and severity counts. Use `--detail` to include vulnerability descriptions and `--json` to output the aggregated result as JSON.

### IAM Policy (for `cage rollout`)

`canarycage` requires several IAM policies to run. Here is a recommended example of IAM policy for `cage rollout` command with Fargate:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": ["ecs:CreateService", "ecs:UpdateService"],
      "Condition": {
        "ArnEquals": {
          "ecs:cluster": "arn:aws:ecs::123456789010:cluster/your-cluster-name"
        },
        "ArnLike": {
          "ecs:task-definition": [
            "arn:aws:ecs::123456789010:task-definition/your-task-definition-name:*"
          ]
        }
      },
      "Effect": "Allow",
      "Resource": [
        "arn:aws:ecs::123456789010:service/your-cluster-name/your-service-name"
      ]
    },
    {
      "Action": ["ecs:DeleteService", "ecs:DescribeServices"],
      "Condition": {
        "ArnEquals": {
          "ecs:cluster": "arn:aws:ecs::123456789010:cluster/your-cluster-name"
        }
      },
      "Effect": "Allow",
      "Resource": [
        "arn:aws:ecs::123456789010:service/your-cluster-name/your-service-name"
      ]
    },
    {
      "Action": ["ecs:RegisterTaskDefinition"],
      "Effect": "Allow",
      "Resource": [
        "arn:aws:ecs::123456789010:task-definition/your-task-definition-name:*"
      ]
    },
    {
      "Action": [
        "ecs:DescribeTasks",
        "ecs:DescribeContainerInstances",
        "ecs:ListTasks",
        "ecs:StopTask"
      ],
      "Condition": {
        "ArnEquals": {
          "ecs:cluster": "arn:aws:ecs::123456789010:cluster/your-cluster-name"
        }
      },
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": ["ecs:RunTask", "ecs:StartTask"],
      "Condition": {
        "ArnEquals": {
          "ecs:cluster": "arn:aws:ecs::123456789010:cluster/your-cluster-name"
        }
      },
      "Effect": "Allow",
      "Resource": [
        "arn:aws:ecs::123456789010:task-definition/your-task-definition-name:*"
      ]
    },
    {
      "Action": ["ecs:DescribeTaskDefinition"],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": [
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticloadbalancing:DescribeTargetHealth",
        "elasticloadbalancing:DescribeTargetGroupAttributes"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": [
        "elasticloadbalancing:RegisterTargets",
        "elasticloadbalancing:DeregisterTargets"
      ],
      "Effect": "Allow",
      "Resource": [
        "arn:aws:elasticloadbalancing::123456789010:targetgroup/your-target-group-name/your-target-group-id"
      ]
    },
    {
      "Action": ["ec2:DescribeSubnets", "ec2:DescribeInstances"],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": ["iam:PassRole"],
      "Condition": {
        "StringEquals": {
          "iam:PassedToService": "ecs-tasks.amazonaws.com"
        }
      },
      "Effect": "Allow",
      "Resource": [
        "arn:aws:iam::123456789010:role/your-task-execution-role",
        "arn:aws:iam::123456789010:role/your-task-role"
      ]
    }
  ]
}
```

### IAM Policy (for `cage audit`)

Here is a recommended example of IAM policy for `cage audit` command:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": ["ecr:BatchGetImage", "ecr:DescribeImageScanFindings"],
      "Effect": "Allow",
      "Resource": ["arn:aws:ecr::123456789010:repository/your-repository-name"]
    },
    {
      "Action": ["ecs:DescribeServices", "ecs:DescribeTaskDefinition"],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

You can restrict the `Resource` field of `ecs:DescribeServices` and `ecs:DescribeTaskDefinition` actions to specific services and task definitions if you want to limit the scope of the policy.

### Why we need `canarycage`

Currently, AWS ECS provides several ways to deploy task to service. [DeploymentCircuitBreaker](https://docs.aws.amazon.com/ja_jp/AmazonECS/latest/APIReference/API_DeploymentCircuitBreaker.html) is one of the choices. This feature is useful for preventing service from being unavailable during deployment. However, it is not enough for us. We need more robust and reliable deployment tool. `canarycage` is designed to be a tool that can deploy tasks to services with high availability.

DeploymentCircuitBreaker automatically detects the failure of the deployment and rolls back the deployment to previosly stable state. During the deployment, the service will be unavailable for a while. We needs a single canary task to check the health of the new task definition before updating the service.

This approach is very robust and reliable. For past 5 years, we have been using this tool for all production microservices running on ECS Fargate with no downtime caused by deployment. Many misconfigurations and bugs have been detected by canary task before updating the service.

## GitHub Actions

Canarycage Github Actions are now unavailable due to security concerns.

## Licence

[MIT](https://github.com/loilo-inc/canarycage/LICENCE)
