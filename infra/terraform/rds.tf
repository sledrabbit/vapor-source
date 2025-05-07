data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ssm_parameter" "db_password_param" {
  name            = "/vapor-server/database/password"
  with_decryption = true
}

resource "aws_security_group" "rds_sg" {
  name        = "${var.db_name}-sg"
  description = "Allow PostgreSQL access"
  vpc_id      = data.aws_vpc.default.id

  # Ingress rule added from ecs.tf
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name      = "${var.db_name}-sg"
    ManagedBy = "Terraform"
  }
}

# temporary for testing
resource "aws_db_parameter_group" "vapor_db_custom_pg" {
  name   = "${lower(var.db_name)}-custom-pg"
  family = "postgres15"

  parameter {
    name         = "rds.force_ssl"
    value        = "0"
    apply_method = "immediate"
  }

  tags = {
    Name      = "${lower(var.db_name)}-custom-pg"
    ManagedBy = "Terraform"
  }
}

resource "aws_db_instance" "vapor_db" {
  identifier             = lower(var.db_name)
  allocated_storage      = 20
  engine                 = "postgres"
  engine_version         = "15"
  instance_class         = var.db_instance_class
  db_name                = var.db_name
  username               = var.db_master_username
  password               = data.aws_ssm_parameter.db_password_param.value
  parameter_group_name   = aws_db_parameter_group.vapor_db_custom_pg.name
  db_subnet_group_name   = aws_db_subnet_group.default_subnets.name
  vpc_security_group_ids = [aws_security_group.rds_sg.id]
  apply_immediately      = true

  skip_final_snapshot = true
  publicly_accessible = false

  tags = {
    Name      = var.db_name
    ManagedBy = "Terraform"
  }
}

resource "aws_db_subnet_group" "default_subnets" {
  name       = "${var.db_name}-subnet-group"
  subnet_ids = data.aws_subnets.default.ids

  tags = {
    Name      = "${var.db_name}-subnet-group"
    ManagedBy = "Terraform"
  }
}

# --- Outputs ---
output "rds_sg_id" {
  description = "ID of the RDS Security Group"
  value       = aws_security_group.rds_sg.id
}
