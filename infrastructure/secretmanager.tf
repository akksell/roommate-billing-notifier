resource "google_secret_manager_secret" "gmail_inbox_user" {
  project   = var.google_project_id
  secret_id = "gmail-inbox-user"

  replication {
    auto {}
  }
}
