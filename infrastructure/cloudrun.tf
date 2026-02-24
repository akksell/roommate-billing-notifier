/*
resource "google_cloud_run_v2_service" "rbn" {
  name     = "rbn"
  location = var.google_project_region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    containers {
      image = var.cloud_run_image
      ports {
        container_port = 8080
      }
      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
      env {
        name  = "PORT"
        value = "8080"
      }
      env {
        name  = "FIRESTORE_PROJECT_ID"
        value = var.google_project_id
      }
      dynamic "env" {
        for_each = var.gmail_inbox_user != "" ? [1] : []
        content {
          name  = "GMAIL_INBOX_USER"
          value = var.gmail_inbox_user
        }
      }
      dynamic "env" {
        for_each = var.gmail_topic_name != "" ? [1] : []
        content {
          name  = "GMAIL_TOPIC_NAME"
          value = var.gmail_topic_name
        }
      }
    }
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

# Allow unauthenticated invocations so Pub/Sub push can reach the service.
# For production, consider restricting to the Pub/Sub subscription identity.
data "google_iam_policy" "run_invoker" {
  binding {
    role = "roles/run.invoker"
    members = [
      "allUsers"
    ]
  }
}

resource "google_cloud_run_v2_service_iam_policy" "rbn" {
  name        = google_cloud_run_v2_service.rbn.name
  location    = google_cloud_run_v2_service.rbn.location
  policy_data = data.google_iam_policy.run_invoker.policy_data
}
*/
