# resource "aws_instance" "web" {
#   ami           = data.aws_ami.ubuntu.id
#   instance_type = var.ec2_instance_type

#   tags = {
#     Name = "HelloWorld"
#   }

#   # From https://aws.amazon.com/ec2/spot/pricing/
#   #
#   # We recommend that you launch your T4g or T3 Spot Instances in
#   # standard mode to avoid paying higher costs.
#   #
#   # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/burstable-performance-instances-unlimited-mode-concepts.html#unlimited-mode-surplus-credits
#   # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-spot-limits.html

#   credit_specification {
#     cpu_credits = var.ec2_cpu_credits
#   }
# }

resource "aws_launch_template" "netbsd-ami" {
  name          = "netbsd-ami-builder"
  instance_type = var.ec2_instance_type

  # Map a device for us to create an image from
  block_device_mappings {
  }

  instance_market_options {
    market_type = "spot"
  }

  # From https://aws.amazon.com/ec2/spot/pricing/
  #
  # We recommend that you launch your T4g or T3 Spot Instances in
  # standard mode to avoid paying higher costs.
  #
  # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/burstable-performance-instances-unlimited-mode-concepts.html#unlimited-mode-surplus-credits
  # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-spot-limits.html

  credit_specification {
    cpu_credits = var.ec2_cpu_credits
  }

  iam_instance_profile {
    arn  = ""
    name = ""
  }

  tag_specifications {
    tags = {
      CodeDeploy = "netbsd-ami"
    }
  }
}

# resource "aws_autoscaling_group" "netbsd-ami" {
#   name                      = "netbsd-ami-builders"
#   max_size                  = 1
#   min_size                  = 0
#   health_check_grace_period = 300 # Shorten this ???
#   health_check_type         = "EC2"
#   vpc_zone_identifier       = [aws_subnet.example1.id, aws_subnet.example2.id]

#   launch_template {
#     id = aws_launch_template.netbsd-ami.id
#   }

#   initial_lifecycle_hook {
#     name                 = "foobar"
#     default_result       = "CONTINUE"
#     heartbeat_timeout    = 2000
#     lifecycle_transition = "autoscaling:EC2_INSTANCE_LAUNCHING"

#     notification_metadata = jsonencode({
#       foo = "bar"
#     })

#     notification_target_arn = "arn:aws:sqs:us-east-1:444455556666:queue1*"
#     role_arn                = "arn:aws:iam::123456789012:role/S3Access"
#   }

#   tag {
#     key                 = "foo"
#     value               = "bar"
#     propagate_at_launch = true
#   }

#   timeouts {
#     delete = "15m"
#   }

#   tag {
#     key                 = "lorem"
#     value               = "ipsum"
#     propagate_at_launch = false
#   }
# }
