/*
output "cloud_run_url" {
  value       = google_cloud_run_v2_service.rbn.uri
  description = "Cloud Run service URL. Set as cloud_run_url (e.g. in tfvars) and re-apply to create the Pub/Sub push subscription."
}

output "gmail_topic" {
  value       = google_pubsub_topic.gmail.name
  description = "Pub/Sub topic name for Gmail watch (use full path: projects/PROJECT/topics/NAME for users.watch)."
}
*/
