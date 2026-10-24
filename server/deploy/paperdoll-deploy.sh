#!/usr/bin/env bash
#
# Deploys the newest server release published by server-deploy.yaml.
#
# Run periodically by paperdoll-deploy.timer. Each run:
#
#   1. Lists the repo's GitHub Releases and picks the newest server-* one.
#   2. Exits if it isn't newer than the deployed version, or if it already
#      failed its health check once (see FAILED_DIR below).
#   3. Downloads the binary and its Sigstore bundle, and verifies the bundle
#      attests that server-deploy.yaml built the binary from that very tag on a
#      GitHub-hosted runner. Nothing is deployed if verification fails.
#   4. Points the `current` symlink at the new release and restarts
#      paperdoll.target.
#   5. Waits until /health reports the new version. If it doesn't, points
#      `current` back at the previous release, restarts again, and marks the
#      new version as failed so later runs don't retry it.
#
# Needs bash, curl, jq, flock, and gh >= 2.93.0. No GitHub credentials are
# needed: releases are public and the bundle is verified offline (gh only
# fetches Sigstore's public trust root).
#
# See server/deploy/README.md for the VPS setup.

set -euo pipefail
# Otherwise a failure inside $(...), e.g. in latest_version, goes unnoticed.
shopt -s inherit_errexit

readonly REPO="${PAPERDOLL_REPO:-fujidaiti/paperdoll}"
readonly ROOT="${PAPERDOLL_ROOT:-/opt/paperdoll}"
readonly STATE_DIR="${PAPERDOLL_DEPLOY_STATE_DIR:-/var/lib/paperdoll-deploy}"
readonly HEALTH_URL="${PAPERDOLL_HEALTH_URL:-http://127.0.0.1:8080/health}"
readonly HEALTH_TIMEOUT_SEC="${PAPERDOLL_HEALTH_TIMEOUT_SEC:-60}"
readonly KEEP_RELEASES="${PAPERDOLL_KEEP_RELEASES:-5}"

readonly ASSET=paperdoll-server-linux-amd64
readonly SIGNER_WORKFLOW="$REPO/.github/workflows/server-deploy.yaml"
readonly TARGET=paperdoll.target
readonly UNITS=(paperdoll-api.service paperdoll-scheduler.service)
readonly RELEASES_DIR="$ROOT/releases"
readonly CURRENT="$ROOT/current"
# One empty file per version that failed its health check.
readonly FAILED_DIR="$STATE_DIR/failed"

# Same scheme as tool/version.sh: <major>.<yyyymmdd>.<HHMMSS>[.beta].
readonly VERSION_NAME_RE='^([0-9]+)\.([0-9]{8})\.([0-9]{6})(\.beta)?$'

log() { echo "$*" >&2; }
die() { log "$*"; exit 1; }

# Prints a key that sorts (as a string) in the same order as the versions do,
# or fails if the version name is malformed. Validating here also keeps the
# name safe to use as a path component.
version_key() {
  [[ "$1" =~ $VERSION_NAME_RE ]] || return 1
  printf '%010d%s%s\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}"
}

# Prints the deployed version, or nothing on a fresh install.
deployed_version() {
  [[ -L "$CURRENT" ]] || return 0
  basename "$(readlink "$CURRENT")"
}

# Prints the newest published server-* version, ignoring malformed tags.
latest_version() {
  local etag="$STATE_DIR/releases.etag" cache="$STATE_DIR/releases.json"
  local body="$STATE_DIR/releases.json.new" status
  # Without the cached body, a 304 would leave nothing to read.
  [[ -f "$cache" ]] || rm -f "$etag"
  # A conditional request answered with 304 doesn't count toward the API's
  # rate limit for unauthenticated clients (60 requests/hour).
  status=$(curl -sS --fail-with-body -o "$body" -w '%{http_code}' \
    --etag-compare "$etag" --etag-save "$etag" \
    -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/$REPO/releases?per_page=30")
  if [[ "$status" == 200 ]]; then
    mv "$body" "$cache"
  else
    rm -f "$body"
  fi

  local best="" best_key="" name key
  while IFS= read -r name; do
    key=$(version_key "$name") || continue
    if [[ "$key" > "$best_key" ]]; then
      best="$name" best_key="$key"
    fi
  done < <(jq -r '.[] | select(.draft | not) | .tag_name | select(startswith("server-")) | ltrimstr("server-")' "$cache")
  echo "$best"
}

download_and_verify() {
  local version="$1" dir="$RELEASES_DIR/$1" tmp
  if [[ -x "$dir/app" ]]; then
    return 0
  fi
  # Left behind if this fails; main() clears it on the next run.
  tmp=$(mktemp -d "$RELEASES_DIR/.download.XXXXXX")

  local base="https://github.com/$REPO/releases/download/server-$version"
  curl -sSfL -o "$tmp/app" "$base/$ASSET"
  curl -sSfL -o "$tmp/app.sigstore.json" "$base/$ASSET.sigstore.json"

  gh attestation verify "$tmp/app" \
    --bundle "$tmp/app.sigstore.json" \
    --repo "$REPO" \
    --signer-workflow "$SIGNER_WORKFLOW" \
    --source-ref "refs/tags/server-$version" \
    --deny-self-hosted-runners \
    || die "Attestation verification failed for $version; not deploying."

  chmod 0755 "$tmp/app"
  rm -rf "$dir"
  mv -T "$tmp" "$dir"
}

# Atomically points `current` at the given release.
switch_to() {
  ln -sfn "releases/$1" "$CURRENT.tmp"
  mv -T "$CURRENT.tmp" "$CURRENT"
}

# Succeeds once /health reports the given version and every unit is active.
wait_healthy() {
  local version="$1" deadline=$((SECONDS + HEALTH_TIMEOUT_SEC)) reported
  while ((SECONDS < deadline)); do
    reported=$(curl -sSf --max-time 2 "$HEALTH_URL" 2> /dev/null | jq -r '.version' 2> /dev/null || true)
    if [[ "$reported" == "$version" ]] && all_active; then
      return 0
    fi
    sleep 2
  done
  return 1
}

# `systemctl is-active` with several units succeeds if any one is active.
all_active() {
  local unit
  for unit in "${UNITS[@]}"; do
    systemctl is-active --quiet "$unit" || return 1
  done
}

# A failed start is caught by wait_healthy, so it mustn't abort the script
# before it can roll back.
restart() {
  systemctl restart "$TARGET" || log "systemctl restart $TARGET failed."
}

# Deletes all but the newest KEEP_RELEASES releases, never the deployed one.
prune() {
  local deployed="$1" dir name key
  local -a names=()
  for dir in "$RELEASES_DIR"/*/; do
    name=$(basename "$dir")
    if key=$(version_key "$name"); then
      names+=("$key $name")
    fi
  done
  ((${#names[@]} > KEEP_RELEASES)) || return 0
  printf '%s\n' "${names[@]}" | sort -r | tail -n "+$((KEEP_RELEASES + 1))" |
    while read -r _ name; do
      [[ "$name" == "$deployed" ]] || rm -rf "${RELEASES_DIR:?}/$name"
    done
}

main() {
  mkdir -p "$RELEASES_DIR" "$FAILED_DIR"
  exec 9> "$STATE_DIR/lock"
  flock -n 9 || die "Another deploy is running."
  rm -rf "$RELEASES_DIR"/.download.*

  local latest deployed
  latest=$(latest_version)
  deployed=$(deployed_version)
  [[ -n "$latest" ]] || die "No server release found."

  if [[ -n "$deployed" ]] && ! [[ "$(version_key "$latest")" > "$(version_key "$deployed")" ]]; then
    log "Up to date ($deployed)."
    return 0
  fi
  if [[ -e "$FAILED_DIR/$latest" ]]; then
    log "Skipping $latest: it failed its health check before. Remove $FAILED_DIR/$latest to retry."
    return 0
  fi

  log "Deploying $latest (currently ${deployed:-none})..."
  download_and_verify "$latest"
  switch_to "$latest"
  restart

  if wait_healthy "$latest"; then
    log "Deployed $latest."
    prune "$latest"
    return 0
  fi

  touch "$FAILED_DIR/$latest"
  if [[ -z "$deployed" ]]; then
    die "$latest failed its health check and there is no previous release to roll back to."
  fi
  log "$latest failed its health check; rolling back to $deployed."
  switch_to "$deployed"
  restart
  if wait_healthy "$deployed"; then
    die "Rolled back to $deployed."
  fi
  die "Rollback to $deployed is unhealthy too."
}

main "$@"
