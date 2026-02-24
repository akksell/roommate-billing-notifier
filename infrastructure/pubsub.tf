/*
resource "google_pubsub_topic" "gmail" {
  name = var.gmail_topic_name
}

# Allow Gmail to publish to the topic
resource "google_pubsub_topic_iam_member" "gmail_push" {
  topic  = google_pubsub_topic.gmail.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:gmail-api-push@system.gserviceaccount.com"
}

resource "google_pubsub_subscription" "gmail_push" {
  name   = "gmail-push"
  topic  = google_pubsub_topic.gmail.name
  push_config {
    push_endpoint = "${var.cloud_run_url}/push"
  }
}
*/
