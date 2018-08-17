resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
  enable_dns_hostnames = true
  tags {
    Name = "cage-test-vpc"
    Group = "canarycage"
  }
}
resource "aws_main_route_table_association" "main" {
  route_table_id = "${aws_route_table.main.id}"
  vpc_id = "${aws_vpc.test.id}"
}

resource "aws_route_table" "main" {
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "cage-test-rtb"
    Group = "canarycage"
  }
}

resource "aws_route" "main" {
  route_table_id = "${aws_route_table.main.id}"
  gateway_id = "${aws_internet_gateway.test.id}"
  destination_cidr_block = "0.0.0.0/0"
}

resource "aws_internet_gateway" "test" {
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "cage-test-igw"
    Group = "canarycage"
  }
}
resource "aws_network_acl" "test" {
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "cage-test-acl"
    Group = "canarycage"
  }
}
resource "aws_subnet" "public_a" {
  cidr_block = "10.0.0.0/20"
  vpc_id = "${aws_vpc.test.id}"
  availability_zone = "us-west-2a"
  tags {
    Name = "cage-test-public-subnet-a"
    Group = "canarycage"
  }
}
resource "aws_subnet" "public_b" {
  cidr_block = "10.0.16.0/20"
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "cage-test-public-subnet-b"
    Group = "canarycage"
  }
}

resource "aws_security_group" "public" {
  name = "cage-sg-test-public"
  vpc_id = "${aws_vpc.test.id}"
  ingress {
    from_port = 80
    protocol = "tcp"
    to_port = 80
    cidr_blocks = [
      "0.0.0.0/0"]
  }
  egress {
    from_port = 0
    protocol = "-1"
    to_port = 0
    cidr_blocks = [
      "0.0.0.0/0"]
  }
  tags {
    Group = "canarycage"
  }
}
resource "aws_ecs_cluster" "test" {
  name = "cage-test"
}

resource "aws_alb" "test" {
  name = "cage-alb-test"
  subnets = [
    "${aws_subnet.public_a.id}",
    "${aws_subnet.public_b.id}"]
  security_groups = [
    "${aws_security_group.public.id}"]
  tags {
    Group = "canarycage"
  }
}

resource "aws_alb_target_group" "test" {
  name = "cage-tg-test"
  port = 80
  protocol = "HTTP"
  vpc_id = "${aws_vpc.test.id}"
  health_check {
    healthy_threshold = 2
    unhealthy_threshold = 2
    timeout = 45
    path = "/health_check"
    interval = 60
  }
  target_type = "ip"
  tags {
    Group = "canarycage"
  }
}

resource "aws_alb_listener" "http" {
  "default_action" {
    target_group_arn = "${aws_alb_target_group.test.arn}"
    type = "forward"
  }
  load_balancer_arn = "${aws_alb.test.arn}"
  port = 80
}

resource "aws_cloudwatch_log_group" "test" {
  name = "cage/test"
  tags {
    Group = "canarycage"
  }
}

resource "aws_iam_role" "http_server_task_role" {
  assume_role_policy = <<-EOS
  {
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
  }
  EOS
}

resource "aws_iam_role_policy_attachment" "http_server_task_role_policy" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
  role = "${aws_iam_role.http_server_task_role.id}"
}

# task execution role
resource "aws_iam_role" "task_execution_role" {
  assume_role_policy = <<-EOS
  {
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
  }
  EOS
}

resource "aws_iam_role_policy_attachment" "task_execution_role_policy" {
  role = "${aws_iam_role.task_execution_role.id}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role" "cage" {
  assume_role_policy = <<-EOS
  {
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
  }
  EOS
}

resource "aws_iam_role_policy_attachment" "cage_policy_ecs" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonECS_FullAccess"
  role = "${aws_iam_role.cage.id}"
}
resource "aws_iam_role_policy_attachment" "cage_policy_ec2" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
  role = "${aws_iam_role.cage.id}"
}
resource "aws_iam_role_policy" "cage_policy" {
  policy = <<-EOS
{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Action":[
            "cloudwatch:GetMetricStatistics",
            "logs:*"
         ],
         "Effect":"Allow",
         "Resource":"*"
      }
   ]
}
  EOS
  role = "${aws_iam_role.cage.id}"
}

variable "test-image-tag" {
  default = "latest"
}
variable "test-container-mode" {
  default = "healthy"
}

locals {
  service_name = "test-http-server"
  container_name = "http-server"
  container_port = 80
}

module "test-server-task-definition-healthy" {
  source = "./module"
  container_name = "${local.container_name}"
  container_port = "${local.container_port}"
  task_family = "cage-test-server-healthy"
  log_group = "${aws_cloudwatch_log_group.test.name}"
  execution_role_arn = "${aws_iam_role.task_execution_role.arn}"
  test_container_mode = "healthy"
  task_role_arn = "${aws_iam_role.http_server_task_role.arn}"
}
module "test-server-task-definition-unhealthy" {
  source = "./module"
  container_name = "${local.container_name}"
  container_port = "${local.container_port}"
  task_family = "cage-test-server-unhealthy"
  log_group = "${aws_cloudwatch_log_group.test.name}"
  execution_role_arn = "${aws_iam_role.task_execution_role.arn}"
  test_container_mode = "unhealthy"
  task_role_arn = "${aws_iam_role.http_server_task_role.arn}"
}

module "test-server-task-definition-up-but-buggy" {
  source = "./module"
  container_name = "${local.container_name}"
  container_port = "${local.container_port}"
  task_family = "cage-test-server-up-but-buggy"
  log_group = "${aws_cloudwatch_log_group.test.name}"
  execution_role_arn = "${aws_iam_role.task_execution_role.arn}"
  test_container_mode = "up-but-buggy"
  task_role_arn = "${aws_iam_role.http_server_task_role.arn}"
}

module "test-server-task-definition-up-but-slow" {
  source = "./module"
  container_name = "${local.container_name}"
  container_port = "${local.container_port}"
  task_family = "cage-test-server-up-but-slow"
  log_group = "${aws_cloudwatch_log_group.test.name}"
  execution_role_arn = "${aws_iam_role.task_execution_role.arn}"
  test_container_mode = "up-but-slow"
  task_role_arn = "${aws_iam_role.http_server_task_role.arn}"
}

module "test-server-task-definition-up-but-exit" {
  source = "./module"
  container_name = "${local.container_name}"
  container_port = "${local.container_port}"
  task_family = "cage-test-server-up-but-exit"
  log_group = "${aws_cloudwatch_log_group.test.name}"
  execution_role_arn = "${aws_iam_role.task_execution_role.arn}"
  test_container_mode = "up-but-exit"
  task_role_arn = "${aws_iam_role.http_server_task_role.arn}"
}

module "test-server-task-definition-up-and-exit" {
  source = "./module"
  container_name = "${local.container_name}"
  container_port = "${local.container_port}"
  task_family = "cage-test-server-up-and-exit"
  log_group = "${aws_cloudwatch_log_group.test.name}"
  execution_role_arn = "${aws_iam_role.task_execution_role.arn}"
  test_container_mode = "up-and-exit"
  task_role_arn = "${aws_iam_role.http_server_task_role.arn}"
}
