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
  "loadBalancerArn": "${aws_alb.test.arn}",
  "currentServiceName": "${aws_ecs_service.test.name}"
}
  EOS
}