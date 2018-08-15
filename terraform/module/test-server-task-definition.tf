variable "container_name" {}
variable "container_port" {}
variable "task_family" {}
variable "test_image_tag" {
  default = "latest"
}
variable "test_container_mode" {}
variable "task_role_arn" {}
variable "execution_role_arn" {}
variable "log_group" {}
resource "aws_ecs_task_definition" "test" {
  container_definitions = <<-EOS
[
    {
        "name": "${var.container_name}",
        "image": "loilodev/http-server:${var.test_image_tag}",
        "cpu": 100,
        "memory": 512,
        "memoryReservation": 500,
        "portMappings": [
            {
                "containerPort": ${var.container_port},
                "hostPort": ${var.container_port}
            }
        ],
        "essential": true,
        "environment": [
          {"name": "PORT", "value": "${var.container_port}"}
        ],
        "command": [
          "${var.test_container_mode}"
        ],
        "logConfiguration": {
            "logDriver": "awslogs",
            "options": {
                "awslogs-group": "${var.log_group}",
                "awslogs-stream-prefix": "cage",
                "awslogs-region": "us-west-2"
            }
        }
    }
]
  EOS
  family = "${var.task_family}"
  task_role_arn = "${var.task_role_arn}"
  execution_role_arn = "${var.execution_role_arn}"
  network_mode = "awsvpc"
  requires_compatibilities = [
    "EC2",
    "FARGATE"
  ]
  cpu = "256"
  memory = "0.5GB"
}

output "task_arn" {
  value = "${aws_ecs_task_definition.test.arn}"
}
output "task_family" {
  value = "${aws_ecs_task_definition.test.family}"
}
output "task_revision" {
  value = "${aws_ecs_task_definition.test.revision}"
}