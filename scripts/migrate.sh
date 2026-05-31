#!/usr/bin/env bash
# Apply migrations against $DB_URL using golang-migrate. Install with:
#   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
set -euo pipefail

DB_URL="${DB_URL:-postgres://chat:chatpw@localhost:5432/chat?sslmode=disable}"
DIR="$(cd "$(dirname "$0")/.." && pwd)"

cmd="${1:-up}"
migrate -path "${DIR}/migrations" -database "${DB_URL}" "${cmd}"
