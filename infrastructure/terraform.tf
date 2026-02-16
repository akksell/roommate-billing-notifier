terraform {
  backend gcs {

  }
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "7.19.0"
    }
  }
}

provider "google" {
  project = var.google_project_id
  region = var.google_project_region
}
