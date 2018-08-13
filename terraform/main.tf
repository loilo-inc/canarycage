resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
  enable_dns_hostnames = true
}
resource "aws_internet_gateway" "test" {
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "cage-test-igw"
  }
}
resource "aws_network_acl" "test" {
  vpc_id = "${aws_vpc.test.id}"
  tags {
    Name = "cage-test-acl"
  }
}
resource "aws_subnet" "public_a" {
  cidr_block = "10.0.0.0/20"
  vpc_id = "${aws_vpc.test.id}"
  availability_zone = "us-west-2a"
}
resource "aws_subnet" "public_b" {
  cidr_block = "10.0.16.0/20"
  vpc_id = "${aws_vpc.test.id}"
}
resource "aws_subnet" "private" {
  cidr_block = "10.0.64.0/20"
  vpc_id = "${aws_vpc.test.id}"
  availability_zone = "us-west-2b"
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
}
resource "aws_security_group" "private" {
  name = "cage-sg-test-private"
  vpc_id = "${aws_vpc.test.id}"
  ingress {
    from_port = 0
    protocol = "-1"
    to_port = 0
    cidr_blocks = [
      "${aws_subnet.public_a.cidr_block}",
      "${aws_subnet.public_b.cidr_block}"
    ]
  }
  egress {
    from_port = 0
    protocol = "-1"
    to_port = 0
    cidr_blocks = [
      "0.0.0.0/0"]
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
  security_groups = ["${aws_security_group.public.id}"]
}

resource "aws_alb_target_group" "blue" {
  name = "cage-tg-test-blue"
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
}

resource "aws_alb_target_group" "green" {
  name = "cage-tg-test-green"
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
}

resource "aws_alb_listener" "blue_http" {
  "default_action" {
    target_group_arn = "${aws_alb_target_group.blue.arn}"
    type = "forward"
  }
  load_balancer_arn = "${aws_alb.test.arn}"
  port = 80
}

resource "aws_alb_listener" "green_http" {
  "default_action" {
    target_group_arn = "${aws_alb_target_group.green.arn}"
    type = "forward"
  }
  load_balancer_arn = "${aws_alb.test.arn}"
  port = 80
}

resource "aws_cloudwatch_log_group" "test" {
  name = "cage/test"
}