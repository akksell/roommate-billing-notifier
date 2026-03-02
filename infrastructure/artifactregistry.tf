resource "google_artifact_registry_repository" "container_image_repository" {
  repository_id = "cloud-run-images-${var.environment}"
  format = "DOCKER"
  description = "Repository for our roommate-billing-notifier-images"
  mode = "STANDARD_REPOSITORY"

  docker_config {
    immutable_tags = true
  }
}