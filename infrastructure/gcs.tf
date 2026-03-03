resource "google_storage_bucket" "app_configuration" {
  name     = "rbn-service-config-${var.environment}"
  location = var.google_project_region
  project  = var.google_project_id

  versioning {
    enabled = true
  }

  uniform_bucket_level_access = true
}
