#!/usr/bin/env bash
# Starts n8n (installed under Node 24) with the Execute Command node enabled:
# n8n 2.x disables it by default, and this pipeline calls the Go tools through it.
source ~/.nvm/nvm.sh >/dev/null && nvm use 24 >/dev/null
export NODES_EXCLUDE='[]' N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true
exec n8n start
