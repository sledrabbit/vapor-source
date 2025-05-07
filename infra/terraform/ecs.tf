# --- ECR ---
resource "aws_ecr_repository" "vapor_server_repo" {
  name                 = var.ecr_repo_name
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name      = var.ecr_repo_name
    ManagedBy = "Terraform"
  }
}

# --- ECS Cluster ---
resource "aws_ecs_cluster" "vapor_cluster" {
  name = var.ecs_cluster_name

  tags = {
    Name      = var.ecs_cluster_name
    ManagedBy = "Terraform"
  }
}

# --- Load Balancer ---
resource "aws_lb" "vapor_server_lb" {
  name               = "${var.ecs_service_name}-lb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.lb_sg.id]
  subnets            = data.aws_subnets.default.ids

  tags = {
    Name      = "${var.ecs_service_name}-lb"
    ManagedBy = "Terraform"
  }
}

resource "aws_security_group" "lb_sg" {
  name        = "${var.ecs_service_name}-lb-sg"
  description = "Allow HTTP traffic to LB"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    protocol    = "tcp"
    from_port   = 80
    to_port     = 80
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name      = "${var.ecs_service_name}-lb-sg"
    ManagedBy = "Terraform"
  }
}

resource "aws_lb_target_group" "vapor_server_tg" {
  name        = "${var.ecs_service_name}-tg"
  port        = var.vapor_server_container_port
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.default.id
  target_type = "ip"

  health_check {
    path                = "/health"
    protocol            = "HTTP"
    matcher             = "200-299"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 3
    unhealthy_threshold = 3
  }

  tags = {
    Name      = "${var.ecs_service_name}-tg"
    ManagedBy = "Terraform"
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.vapor_server_lb.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.vapor_server_tg.arn
  }
}

# --- ECS Task Definition ---

# IAM Role and Policy Updates
resource "aws_iam_role" "ecs_task_execution_role" {
  name = "${var.ecs_task_family}-exec-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
    }]
  })
  tags = { ManagedBy = "Terraform" }
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_policy" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

resource "aws_iam_policy" "ecs_ssm_parameter_access" {
  name        = "${var.ecs_task_family}-ssm-access-policy"
  description = "Allow ECS task execution role to access specific SSM parameters"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter"
        ]
        Effect   = "Allow"
        Resource = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/vapor-server/database/password"
      },
      {
        Action = [
          "kms:Decrypt"
        ]
        Effect   = "Allow"
        Resource = "arn:aws:kms:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:key/*"
        Condition = {
          "StringEquals" = {
            "kms:ViaService" = "ssm.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      }
    ]
  })
  tags = { ManagedBy = "Terraform" }
}

resource "aws_iam_role_policy_attachment" "ecs_ssm_parameter_policy_attachment" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = aws_iam_policy.ecs_ssm_parameter_access.arn
}

resource "aws_ecs_task_definition" "vapor_server_task" {
  family                   = var.ecs_task_family
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.ecs_task_cpu
  memory                   = var.ecs_task_memory
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "ARM64"
  }

  container_definitions = jsonencode([
    {
      name      = var.ecs_service_name
      image     = var.vapor_server_image_uri
      cpu       = var.ecs_task_cpu
      memory    = var.ecs_task_memory
      essential = true
      portMappings = [
        {
          containerPort = var.vapor_server_container_port
          hostPort      = var.vapor_server_container_port
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "PORT", value = tostring(var.vapor_server_container_port) },
        { name = "POSTGRES_USER", value = var.db_master_username },
        { name = "POSTGRES_HOST", value = aws_db_instance.vapor_db.address },
        { name = "POSTGRES_PORT", value = tostring(aws_db_instance.vapor_db.port) },
        { name = "POSTGRES_DB", value = var.db_name },
        { name = "AWS_REGION", value = data.aws_region.current.name },
        { name = "DB_PASSWORD_SSM_PARAM_NAME", value = "/vapor-server/database/password" }
      ]
      secrets = [
        {
          name      = "POSTGRES_PASSWORD"
          valueFrom = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/vapor-server/database/password"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ecs_log_group.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name      = var.ecs_task_family
    ManagedBy = "Terraform"
  }
}

# --- ECS Service ---
resource "aws_ecs_service" "vapor_server_service" {
  name            = var.ecs_service_name
  cluster         = aws_ecs_cluster.vapor_cluster.id
  task_definition = aws_ecs_task_definition.vapor_server_task.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = data.aws_subnets.default.ids
    security_groups  = [aws_security_group.ecs_service_sg.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.vapor_server_tg.arn
    container_name   = var.ecs_service_name
    container_port   = var.vapor_server_container_port
  }

  depends_on = [
    aws_lb_listener.http
  ]

  tags = {
    Name      = var.ecs_service_name
    ManagedBy = "Terraform"
  }
}

# --- Security Groups for ECS ---
resource "aws_security_group" "ecs_service_sg" {
  name        = "${var.ecs_service_name}-sg"
  description = "Allow traffic to the Vapor Server ECS service via LB"
  vpc_id      = data.aws_vpc.default.id

  # Allow ingress ONLY from the Load Balancer Security Group
  ingress {
    from_port       = var.vapor_server_container_port
    to_port         = var.vapor_server_container_port
    protocol        = "tcp"
    security_groups = [aws_security_group.lb_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name      = "${var.ecs_service_name}-sg"
    ManagedBy = "Terraform"
  }
}

# Add rule to RDS Security Group to allow access FROM the ECS Service SG
resource "aws_security_group_rule" "rds_allow_from_ecs" {
  type                     = "ingress"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.ecs_service_sg.id
  security_group_id        = aws_security_group.rds_sg.id
  description              = "Allow PostgreSQL access from ECS Service"
}

# --- CloudWatch Log Group ---
resource "aws_cloudwatch_log_group" "ecs_log_group" {
  name = "/ecs/${var.ecs_task_family}"

  tags = {
    Name      = "/ecs/${var.ecs_task_family}"
    ManagedBy = "Terraform"
  }
}
