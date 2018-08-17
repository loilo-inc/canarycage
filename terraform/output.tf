output "test-server-healthy-td_arn" {
  value = "${module.test-server-task-definition-healthy.task_arn}"
}
output "test-server-unhealthy-td-arn" {
  value = "${module.test-server-task-definition-unhealthy.task_arn}"
}
output "test-server-up-but-buggy-td-arn" {
  value = "${module.test-server-task-definition-up-but-buggy.task_arn}"
}
output "test-server-up-but-slow-td-arn" {
  value = "${module.test-server-task-definition-up-but-slow.task_arn}"
}
output "task-server-up-and-exit-td-arn" {
  value = "${module.test-server-task-definition-up-and-exit.task_arn}"
}
output "task-server-up-but-exit-td-arn" {
  value = "${module.test-server-task-definition-up-but-exit.task_arn}"
}
output "cage.json" {
  value = <<-EOS
{
  "region": "us-east-2",
  "cluster": "${aws_ecs_cluster.test.name}",
  "loadBalancerArn": "${aws_alb.test.arn}"
}
  EOS
}

output "service-template.json" {
  value = <<-EOS
{
  "cluster": "${aws_ecs_cluster.test.name}",
  "serviceName": "service-current",
  "taskDefinition": "",
  "loadBalancers": [
    {
      "targetGroupArn": "${aws_alb_target_group.test.arn}",
      "containerName": "${local.container_name}",
      "containerPort": ${local.container_port}
    }
  ],
  "desiredCount": 2,
  "launchType": "FARGATE",
  "deploymentConfiguration": {
    "maximumPercent": 200,
    "minimumHealthyPercent": 100
  },
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": [
        "${aws_subnet.public_a.id}",
        "${aws_subnet.public_b.id}"
      ],
      "securityGroups": [
        "${aws_security_group.public.id}"
      ],
      "assignPublicIp": "ENABLED"
    }
  },
  "healthCheckGracePeriodSeconds": 0,
  "schedulingStrategy": "REPLICA"
}
  EOS
}