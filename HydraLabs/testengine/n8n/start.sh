#!/usr/bin/env bash
# Starts n8n (installed under Node 24) with the Execute Command node enabled:
# n8n 2.x disables it by default, and this pipeline calls the Go tools through it.
source ~/.nvm/nvm.sh >/dev/null && nvm use 24 >/dev/null
export NODES_EXCLUDE='[]' N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true
# secrets for the pipeline (JIRA_BASE_URL, JIRA_EMAIL, JIRA_TOKEN, HYDRA_CHAT_WEBHOOK) live outside the repo
HERE="$(cd "$(dirname "$0")" && pwd)"
for f in "$HOME/.hydra.env" "$HERE/../../.hydra.env"; do  # home dir, or next to the repo's package.json
  [ -f "$f" ] && set -a && source "$f" && set +a
done
exec n8n start
