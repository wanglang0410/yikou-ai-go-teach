#!/bin/sh
set -e

CONFIG_FILE="/app/config/config-docker.yml"

if [ -n "$AI_API_KEY" ]; then
  sed -i "s|api-key:.*|api-key: ${AI_API_KEY}|" "$CONFIG_FILE"
fi

exec ./server -env=docker
