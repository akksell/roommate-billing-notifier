resource "google_storage_bucket" "backend" {
  name = "billing-notifier-tf-state-${var.environment}"
  project = var.google_project_id
  location = "US"
  storage_class = "STANDARD"
  force_destroy = false
  public_access_prevention = "enforced"

  versioning {
    enabled = true
  }
}

resource "local_file" "default" {
  file_permission = "0644"
  filename        = "config/backend.${var.environment}.config"

  content = templatefile("${path.module}/backend.tftpl", { bucket_name = google_storage_bucket.backend.name })
}