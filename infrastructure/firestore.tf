resource "google_firestore_database" "default" {
  project     = var.google_project_id
  name        = "billing"
  location_id = var.google_project_region
  type        = "FIRESTORE_NATIVE"
  database_edition = "STANDARD"
  delete_protection_state = "DELETE_PROTECTION_ENABLED"
}
