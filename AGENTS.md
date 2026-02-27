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

### Configuration (env vars)

| Variable | Required | Default | Description |
|---|---|---|---|
| `GMAIL_INBOX_USER` | yes | — | Gmail address to read bills from and send notifications as |
| `FIRESTORE_PROJECT_ID` | yes | — | GCP project for Firestore |
| `CONFIG_PATH` | no | — | Path to optional YAML config file (for `filters`) |
| `PORT` | no | `8080` | HTTP listen port |
| `GMAIL_TOPIC_NAME` | no | — | Pub/Sub topic name for Gmail watch |
| `HISTORY_ID_DOC_PATH` | no | `gmail_history` | Firestore doc ID storing last Gmail history ID |
| `CONFIG_COLLECTION` | no | `config` | Firestore collection for app config docs |
| `FILTER_BILLER_SENDERS` | no | — | Comma-separated sender substrings to allow |
| `FILTER_KEYWORDS` | no | — | Comma-separated keywords that must all appear in message |
| `FILTER_LABEL_IDS` | no | — | Comma-separated Gmail label IDs that must all be present |

Filters can also be specified in the YAML config file under a `filters:` key (overrides env vars).

### Firestore data model

- `config/{HISTORY_ID_DOC_PATH}` — `{ historyId: string }` (Gmail sync cursor)
- `roommates/{id}` — `{ email, displayName, active }` (active roommates to split among)
- `bills/{gmailMessageId}` — bill document (idempotent by message ID)
  - `debts/{roommateId}` — `{ roommateId, amount, status, paidAt?, paidBy? }`
- Bill status rolls up: `unpaid` → `partial` → `paid` as debts are marked paid via `POST /bills/{billId}/debts/{roommateId}/paid`

### GCP / Auth

The service uses Application Default Credentials (service account on Cloud Run). The Gmail client uses domain-wide delegation (JWT impersonation) so it can read/send as the configured inbox user.

CI authenticates via Workload Identity Federation (no long-lived keys).
