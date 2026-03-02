variable "google_project_id" {
  type = string
}

variable "google_project_region" {
  type    = string
  default = "us-central1"
}

variable "environment" {
  type = string
}

/*
variable "cloud_run_image" {
  type        = string
  description = "Container image URL for the RBN service (e.g. gcr.io/PROJECT/rbn)"
}

variable "gmail_topic_name" {
  type    = string
  default = "gmail-watch"
}

variable "cloud_run_url" {
  type        = string
  description = "URL of the Cloud Run service (set after first deploy, then create subscription)"
  default     = ""
}

variable "gmail_inbox_user" {
  type        = string
  description = "Gmail inbox user (e.g. billing@example.com). Required; set in tfvars, do not commit."
}
*/