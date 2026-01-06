#!/usr/bin/env bash
# Sends a notification if a new species is detected

# Load centralized configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/config.sh"

# Also source raw config for backwards compatibility (IDFILE, etc.)
source_raw_config

trap 'rm -f $lastcheck' EXIT

lastcheck="$(mktemp)"

[ -f "${IDFILE}" ] || touch "${IDFILE}"

cp "${IDFILE}" "${lastcheck}"

"${SCRIPT_DIR}/update_species.sh"

if ! diff "${IDFILE}" "${lastcheck}" &> /dev/null; then
  SPECIES=$(diff "${IDFILE}" "${lastcheck}" \
    | tail -n+2 |\
    awk '{for(i=2;i<=NF;++i)printf $i""FS ; print ""}' )

  NOTIFICATION="New Species Detection: "${SPECIES[@]}""
  echo "Sending the following notification:
${NOTIFICATION}"

  APPRISE_CONFIG="${BIRDNET_BASE_PATH}/apprise.txt"
  if [ -s "${APPRISE_CONFIG}" ]; then
    "${BIRDNET_BASE_PATH}/birdnet/bin/apprise" -vv -t 'New Species Detected' -b "${NOTIFICATION}" --config="${APPRISE_CONFIG}"
  fi
fi
