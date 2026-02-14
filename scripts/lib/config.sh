#!/usr/bin/env bash
# DEPRECATED: This script is superseded by internal/config/ in the Go rewrite.
# It is kept for reference only. Use the Go configuration package for new code.
# =============================================================================
# BirdNET-Pi Configuration Helper
# =============================================================================
#
# Provides centralized configuration loading with environment variable support.
# Source this file to get configuration functions and computed paths.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/config.sh"
#   my_value=$(get_config "CONFIDENCE")
#
# Environment Override:
#   BIRDNET_<KEY> overrides <KEY> from config file
#   Example: export BIRDNET_RECS_DIR=/mnt/external/BirdSongs
#
# =============================================================================

# Configuration file path
BIRDNET_CONFIG_FILE="${BIRDNET_CONFIG_FILE:-/etc/birdnet/birdnet.conf}"

# Base paths computed relative to this script
# scripts/lib/config.sh -> scripts/lib -> scripts -> project root
BIRDNET_BASE_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Flag to track if config has been loaded
_BIRDNET_CONFIG_LOADED=0

# Associative array to cache config values (bash 4+)
declare -A _BIRDNET_CONFIG_CACHE 2>/dev/null || true

# =============================================================================
# Core Functions
# =============================================================================

# Load configuration file into cache
# This is called automatically by get_config() if needed
load_birdnet_config() {
    local force_reload="${1:-0}"

    # Skip if already loaded unless force reload
    if [[ "$_BIRDNET_CONFIG_LOADED" -eq 1 && "$force_reload" -eq 0 ]]; then
        return 0
    fi

    # Clear cache
    _BIRDNET_CONFIG_CACHE=()

    # Load from config file if it exists
    if [[ -f "$BIRDNET_CONFIG_FILE" ]]; then
        # Read config file, skipping comments and empty lines
        while IFS='=' read -r key value || [[ -n "$key" ]]; do
            # Skip comments and empty lines
            [[ "$key" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$key" ]] && continue

            # Trim whitespace
            key=$(echo "$key" | xargs)
            value=$(echo "$value" | xargs)

            # Strip surrounding quotes from value
            value="${value#\"}"
            value="${value%\"}"
            value="${value#\'}"
            value="${value%\'}"

            # Store in cache
            _BIRDNET_CONFIG_CACHE[$key]="$value"
        done < "$BIRDNET_CONFIG_FILE"
    fi

    _BIRDNET_CONFIG_LOADED=1
}

# Get configuration value with environment override support
#
# Priority (highest to lowest):
# 1. Environment variable BIRDNET_<KEY>
# 2. Config file value
# 3. Default value (second argument)
#
# Usage:
#   value=$(get_config "CONFIDENCE" "0.7")
#   value=$(get_config "RECS_DIR")
#
get_config() {
    local key="$1"
    local default="${2:-}"

    # Ensure config is loaded
    load_birdnet_config

    # Check for environment override first (BIRDNET_<KEY>)
    local env_key="BIRDNET_${key}"
    local env_value="${!env_key:-}"

    if [[ -n "$env_value" ]]; then
        echo "$env_value"
        return 0
    fi

    # Check cached config value
    if [[ -v "_BIRDNET_CONFIG_CACHE[$key]" ]]; then
        echo "${_BIRDNET_CONFIG_CACHE[$key]}"
        return 0
    fi

    # Return default
    echo "$default"
}

# Force reload configuration from disk
reload_birdnet_config() {
    load_birdnet_config 1
}

# =============================================================================
# Computed Path Variables
# =============================================================================

# These are set when config.sh is sourced
# They respect BIRDNET_* environment overrides

get_birdnet_paths() {
    # Base project paths (always computed, not from config)
    export BIRDNET_BASE_PATH
    export BIRDNET_DB_PATH="${BIRDNET_BASE_PATH}/data/db/birds.db"
    export BIRDNET_MODEL_PATH="${BIRDNET_BASE_PATH}/model"
    export BIRDNET_FONT_PATH="${BIRDNET_BASE_PATH}/data/fonts"
    export BIRDNET_SCHEMA_PATH="${BIRDNET_BASE_PATH}/data/config_schema.json"

    # User-configurable paths from config
    export BIRDNET_USER=$(get_config "BIRDNET_USER" "pi")
    export BIRDNET_HOME="/home/${BIRDNET_USER}"
    export BIRDNET_RECS_DIR=$(get_config "RECS_DIR" "${BIRDNET_HOME}/BirdSongs")
    export BIRDNET_EXTRACTED_DIR="${BIRDNET_RECS_DIR}/Extracted"
    export BIRDNET_STREAMDATA_DIR="${BIRDNET_RECS_DIR}/StreamData"
    export BIRDNET_ANALYZING_NOW="${BIRDNET_STREAMDATA_DIR}/analyzing_now.txt"
}

# =============================================================================
# Backwards Compatibility
# =============================================================================

# Source the original config file for scripts that expect raw variables
# This is a convenience for migration - new scripts should use get_config()
source_raw_config() {
    if [[ -f "$BIRDNET_CONFIG_FILE" ]]; then
        # shellcheck source=/dev/null
        source "$BIRDNET_CONFIG_FILE"
    fi
}

# =============================================================================
# Initialize on source
# =============================================================================

# Load config and set paths when this file is sourced
load_birdnet_config
get_birdnet_paths
