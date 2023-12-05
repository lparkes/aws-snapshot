resource "aws_codedeploy_app" "netbsd-ami" {
  compute_platform = "Server"
  name             = "netbsd-ami"
}

resource "aws_codedeploy_deployment_config" "netbsd-ami-builder" {
  deployment_config_name = "netbsd-ami-builder"
  compute_platform       = "Server"
}

resource "aws_codedeploy_deployment_group" "netbsd-ami" {
  app_name              = aws_codedeploy_app.netbsd-ami.name
  deployment_group_name = "netbsd-ami-group"
  service_role_arn      = aws_iam_role.netbsd-ami.arn

  ec2_tag_set {
    ec2_tag_filter {
      type  = "KEY_AND_VALUE"
      key   = "CodeDeploy"
      value = "netbsd-ami"
    }
  }

  # trigger_configuration {
  #   trigger_events     = ["DeploymentFailure"]
  #   trigger_name       = "example-trigger"
  #   trigger_target_arn = aws_sns_topic.example.arn
  # }

  # auto_rollback_configuration {
  #   enabled = true
  #   events  = ["DEPLOYMENT_FAILURE"]
  # }

  # alarm_configuration {
  #   alarms  = ["my-alarm-name"]
  #   enabled = true
  # }

  # outdated_instances_strategy = "UPDATE"
}
