resource "google_service_account" "github_actions_infra_ci_account" {
  account_id = "github-actions-infra-mgr"
  display_name = "GitHub Actions GCP Resource Manager"
  description = "Service account used within github actions to manager project infrastructure. Can provision new infra or update existing infra. Should NOT delete existing infra"
}

resource "google_service_account_iam_member" "github_actions_infra_ci_account_impersonation" {
  service_account_id = google_service_account.github_actions_infra_ci_account.id
  role = "roles/iam.workloadIdentityUser"
  # This should have come from a data block but data blocks for wif are in beta for
  # the google terraform provider. Alternatively, it should have been provisioned here
  # instead of the GCP UI
  member = "principalSet://iam.googleapis.com/projects/453933398414/locations/global/workloadIdentityPools/github-actions/attribute.repository_id/1157511442"
}

resource "google_project_iam_member" "gh_actions_cloud_run_member" {
  project = var.google_project_id
  role = "roles/run.admin"
  member = "serviceAccount:${google_service_account.github_actions_infra_ci_account.email}"
}

resource "google_project_iam_member" "gh_actions_artifact_repo_member" {
  project = var.google_project_id
  role = "roles/artifactregistry.admin"
  member = "serviceAccount:${google_service_account.github_actions_infra_ci_account.email}"
}

resource "google_project_iam_member" "gh_actions_pub_sub_member" {
  project = var.google_project_id
  role = "roles/pubsub.admin"
  member = "serviceAccount:${google_service_account.github_actions_infra_ci_account.email}"
}

resource "google_project_iam_member" "gh_actions_firestore_member" {
  project = var.google_project_id
  role = "roles/datastore.owner"
  member = "serviceAccount:${google_service_account.github_actions_infra_ci_account.email}"
}

resource "google_project_iam_member" "gh_actions_storage_bucket_member" {
  project = var.google_project_id
  role = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.github_actions_infra_ci_account.email}"
}