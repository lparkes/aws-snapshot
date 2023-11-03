variable "ec2_instance_type" {
  description = ""
  type        = string
  default     = "t4g.nano"
}

variable "ec2_cpu_credits" {
  description = "AWS says \"We recommend that you launch your T4g or T3 Spot Instances in standard mode to avoid paying higher costs\"."
  type        = string
  default     = "standard"
}
