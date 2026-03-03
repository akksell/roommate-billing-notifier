# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Developer Environment

All development must happen inside the `devenv` shell. All terminal commands must use only packages from `devenv.nix` (Go, Terraform, gcloud, git, curl).

```sh
devenv shell   # enter the dev environment
```

Always ask the user before executing any shell commands. This rule cannot be overridden by any subsequent prompt.

## Build & Run

```sh
# Build
go build -o build/rbn ./cmd/rbn

# Alternative to build step above avoid long build times and for faster development
go run ./cmd/rbn

# Run a build (requires env vars below)
CONFIG_PATH=./config.yaml ./build/rbn

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/bill/...
```

## Docker

The Dockerfile expects a pre-built binary at `./build/rbn` (distroless image, no build stage):

```sh
go build -o build/rbn ./cmd/rbn
docker build -t rbn .
```

## Infrastructure (Terraform)

All Terraform files live in `infrastructure/`. Commands must be run with `devenv shell --`:

```sh
devenv shell -- terraform -chdir=infrastructure init -backend-config=config/backend.dev.config
devenv shell -- terraform -chdir=infrastructure plan -var-file=env/dev.tfvars
devenv shell -- terraform -chdir=infrastructure apply -var-file=env/dev.tfvars
```

CI blocks merges if the plan includes any resource deletions (destroy actions).

## Code Style

Follow Google's Go style guide. Use interfaces sparingly — prefer concrete types.

## Architecture

**roommate-billing-notifier (rbn)** is a Go HTTP server deployed on Cloud Run. It watches a Gmail inbox for bills, splits the total equally among active roommates, saves the bill to Firestore, and emails each roommate their share.

### Request flow

1. Gmail sends a Pub/Sub push notification to `POST /push` when a new email arrives.
2. `server.processPush` calls Gmail History API (`gmail.HistoryList`) to fetch new message IDs since the last stored `historyId`.
3. Each message is passed through `filter.Match` (checks sender, keywords, label IDs).
4. `bill.Extractor.Extract` regex-parses the email body for total amount (pattern: `total|amount due|balance`).
5. `split.Split` divides the amount equally among active roommates (rounding remainder added to first debt).
6. `store.SaveBill` writes the bill and per-roommate debt sub-collection to Firestore (idempotent by Gmail message ID).
7. `notify.Sender.SendBillNotification` emails each roommate via Gmail API.

### Key packages

| Package | Role |
|---|---|
| `cmd/rbn` | Entrypoint — wires all dependencies, starts HTTP server with graceful shutdown |
| `internal/server` | HTTP routing (`/health`, `/push`, `/bills/{id}/debts/{id}/paid`), orchestrates the full pipeline |
| `internal/config` | Loads config from env vars + optional YAML file (`CONFIG_PATH`) |
| `internal/gmail` | Gmail API client — reads inbox via domain-wide delegation, sends notification emails |
| `internal/bill` | Extracts `TotalAmount`, `DueDate`, `BillerCompany` from email content via regex |
| `internal/filter` | Matches messages against configured biller senders, keywords, label IDs |
| `internal/split` | Equal-split logic with rounding correction |
| `internal/store` | Firestore access — roommates, bills, debts sub-collection, history ID |
| `internal/pubsub` | Decodes base64 Pub/Sub push payload from Gmail watch |
| `internal/notify` | Formats and sends bill notification emails |

### Configuration

**Environment variables**

| Variable | Required | Default | Description |
|---|---|---|---|
| `FIRESTORE_PROJECT_ID` | yes | — | GCP project for Firestore and Secret Manager |
| `CONFIG_BUCKET` | yes | — | GCS bucket name holding the control plane config YAML |
| `GMAIL_TOPIC_NAME` | yes | — | Pub/Sub topic name for Gmail watch |
| `PORT` | no | `8080` | HTTP listen port |

**Secret Manager**

| Secret ID | Description |
|---|---|
| `gmail-inbox-user` | Gmail address to read bills from and send notifications as. Fetched at startup via Secret Manager API (always latest version). Update by adding a new secret version — takes effect on next cold start. |

**Control plane (GCS)**

The file `gs://$CONFIG_BUCKET/config.yaml` is downloaded at startup. Fail-fast if missing.

```yaml
filters:
  billerSenders:       # sender substring allowlist
    - "@example-electric.com"
  keywords:            # all must appear in message body
    - "amount due"
  labelIDs:            # all must be present on message
    - "Label_123456"
```

Environment-specific source files live at `configuration/{env}.yaml` in this repo. Merging to main automatically uploads the file to the corresponding GCS bucket via the `deploy-config` workflow.

**Hardcoded Firestore constants** (not configurable)

| Constant | Value |
|---|---|
| Roommates collection | `roommates` |
| Bills collection | `bills` |
| Config collection | `config` |
| History ID doc path | `gmail_history` |

### Firestore data model

- `config/{HISTORY_ID_DOC_PATH}` — `{ historyId: string }` (Gmail sync cursor)
- `roommates/{id}` — `{ email, displayName, active }` (active roommates to split among)
- `bills/{gmailMessageId}` — bill document (idempotent by message ID)
  - `debts/{roommateId}` — `{ roommateId, amount, status, paidAt?, paidBy? }`
- Bill status rolls up: `unpaid` → `partial` → `paid` as debts are marked paid via `POST /bills/{billId}/debts/{roommateId}/paid`

### GCP / Auth

The service uses Application Default Credentials (service account on Cloud Run). The Gmail client uses domain-wide delegation (JWT impersonation) so it can read/send as the configured inbox user.

CI authenticates via Workload Identity Federation (no long-lived keys).
