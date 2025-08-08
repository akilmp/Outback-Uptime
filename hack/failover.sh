#!/usr/bin/env bash
set -euo pipefail

PRIMARY_ENDPOINT=${1:-primary}
SECONDARY_ENDPOINT=${2:-secondary}
RESOURCE_GROUP=${RESOURCE_GROUP:-""}
PROFILE_NAME=${PROFILE_NAME:-""}
DRY_RUN=${DRY_RUN:-false}

if [[ -z "$RESOURCE_GROUP" || -z "$PROFILE_NAME" ]]; then
  echo "RESOURCE_GROUP and PROFILE_NAME must be set" >&2
  exit 1
fi

run() {
  if [[ "$DRY_RUN" == true ]]; then
    echo "DRY RUN: $*" >&2
  else
    "$@"
  fi
}

get_weight() {
  if [[ "$DRY_RUN" == true ]]; then
    if [[ "$1" == "$PRIMARY_ENDPOINT" ]]; then
      echo "${PRIMARY_WEIGHT:-50}"
    else
      echo "${SECONDARY_WEIGHT:-50}"
    fi
  else
    az network traffic-manager endpoint show \
      --name "$1" \
      --resource-group "$RESOURCE_GROUP" \
      --profile-name "$PROFILE_NAME" \
      --type externalEndpoints \
      --query 'properties.weight' -o tsv
  fi
}

set_weight() {
  run az network traffic-manager endpoint update \
    --name "$1" \
    --resource-group "$RESOURCE_GROUP" \
    --profile-name "$PROFILE_NAME" \
    --type externalEndpoints \
    --weight "$2" >/dev/null
}

orig_primary=$(get_weight "$PRIMARY_ENDPOINT")
orig_secondary=$(get_weight "$SECONDARY_ENDPOINT")

rollback() {
  echo "Rolling back weights" >&2
  set_weight "$PRIMARY_ENDPOINT" "$orig_primary"
  set_weight "$SECONDARY_ENDPOINT" "$orig_secondary"
}
trap rollback EXIT

echo "Failing over from $PRIMARY_ENDPOINT to $SECONDARY_ENDPOINT" >&2
set_weight "$PRIMARY_ENDPOINT" 0
set_weight "$SECONDARY_ENDPOINT" 100

new_primary=$(get_weight "$PRIMARY_ENDPOINT")
if [[ "$new_primary" != "0" ]]; then
  echo "Validation failed: expected weight 0 for $PRIMARY_ENDPOINT" >&2
  exit 1
fi
echo "Failover complete" >&2
trap - EXIT
