locals {
  container_name = "http-server"
  container_port = 8000
  task_family = "cage-test"
}
output "ff" {
  value = <<-EOS
{
    "cluster": "${aws_ecs_cluster.test.name}",
    "serviceName": "{{.ServiceName}}",
    "taskDefinition": "{{.TaskDefinitionArn}}",
    "loadBalancers": [
        {
            "targetGroupArn": "{{.TargetGroupArn}}",
            "containerName": "${local.container_name}",
            "containerPort": ${local.container_port}
        }
    ],
    "desiredCount": 1,
    "launchType": "FARGATE",
    "deploymentConfiguration": {
        "maximumPercent": 200,
        "minimumHealthyPercent": 100
    },
    "networkConfiguration": {
        "awsvpcConfiguration": {
            "subnets": [
                "${aws_subnet.private.id}"
            ],
            "securityGroups": [
                "${aws_security_group.private.id}"
            ],
            "assignPublicIp": "ENABLED"
        }
    },
    "healthCheckGracePeriodSeconds": 0
}
  EOS
}

output "fg" {
  value = <<-EOS
{
    "family": "${local.task_family}",
    "taskRoleArn": "arn:aws:iam::828165198279:role/lnsh-dev-ecs-task-role",
    "executionRoleArn": "arn:aws:iam::828165198279:role/lnsh-dev-ecs-task-execution-role",
    "networkMode": "awsvpc",
    "containerDefinitions": [
        {
            "name": "${local.container_name}",
            "image": "loilodev/canarycage:latest",
            "cpu": 100,
            "memory": 512,
            "memoryReservation": 500,
            "portMappings": [
                {
                    "containerPort": ${local.container_port},
                    "hostPort": ${local.container_port}
                }
            ],
            "essential": true,
            "environment": [
                {"name": "NODE_ENV", "value": "development"}
            ],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-group": "${aws_cloudwatch_log_group.test.name}",
                    "awslogs-stream-prefix": "cage",
                    "awslogs-region": "us-west-2"
                }
            }
        }
    ],
    "requiresCompatibilities": [
        "EC2", "FARGATE"
    ],
    "cpu": "256",
    "memory": "0.5GB"
}
  EOS
}