#!/usr/bin/env bash
# Update the species list
#set -x

# Load centralized configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/config.sh"

# Also source raw config for backwards compatibility (IDFILE, etc.)
source_raw_config

if [ -f "${BIRDNET_DB_PATH}" ]; then
    sqlite3 "${BIRDNET_DB_PATH}" "SELECT DISTINCT(Com_Name) FROM detections" | sort > "${IDFILE}"
fi
