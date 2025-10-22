#!/usr/bin/env bash
set -euo pipefail

# Creates a local ./secrets directory and placeholder secret files
# Intended to be run on the target host (e.g. your VPS) from the project root.

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SECRETS_DIR="$ROOT_DIR/secrets"

mkdir -p "$SECRETS_DIR"
echo "Created/verified secrets directory: $SECRETS_DIR"

create_if_missing() {
  local name="$1"
  local path="$SECRETS_DIR/$name"
  if [ -f "$path" ]; then
    echo "Skipping existing secret file: $path"
  else
    echo "change-me" > "$path"
    chmod 600 "$path"
    echo "Created placeholder secret: $path (chmod 600)"
  fi
}

create_if_missing whatsapp_token
create_if_missing whatsapp_app_secret
create_if_missing meta_app_secret
create_if_missing admin_password

echo
echo "Place real secret values in files under $SECRETS_DIR or create Swarm secrets from these files:" 
echo "  docker secret create whatsapp_token $SECRETS_DIR/whatsapp_token"
echo
echo "Note: these files are gitignored by default; do NOT commit real secrets to your repo."
