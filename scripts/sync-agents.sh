#!/usr/bin/env bash
# Sync the data/agents/ mirror from the cofiswarm-config SoT (config/agents/).
#
# The registry's data/agents/ is a *mirror*; cofiswarm-config is the source of
# truth. Without a sync step the two silently drift (e.g. a new agent lands in
# the SoT but never reaches the mirror). This keeps them byte-identical.
#
# Usage:
#   scripts/sync-agents.sh            copy SoT -> mirror (add / update / prune)
#   scripts/sync-agents.sh --check    report drift, write nothing, non-zero on drift
#
# SoT location: $COFISWARM_CONFIG_REPO (default ../cofiswarm-config). In --check
# mode a missing SoT is a clean SKIP (exit 0) so per-repo CI — which checks out
# only this repo — stays green; a plain sync with no SoT fails loudly.
set -euo pipefail
shopt -s nullglob

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_REPO="${COFISWARM_CONFIG_REPO:-${ROOT}/../cofiswarm-config}"
SRC="${CONFIG_REPO}/config/agents"
DST="${ROOT}/data/agents"

check=0
[[ "${1:-}" == "--check" ]] && check=1

if [[ ! -d "$SRC" ]]; then
  if [[ $check -eq 1 ]]; then
    echo "skip: SoT agents dir not found ($SRC) — set COFISWARM_CONFIG_REPO to enable the drift check"
    exit 0
  fi
  echo "error: SoT agents dir not found: $SRC (set COFISWARM_CONFIG_REPO)" >&2
  exit 2
fi
mkdir -p "$DST"

diffs=0
# add / update: every SoT file must exist byte-identical in the mirror
for f in "$SRC"/*.json; do
  name="$(basename "$f")"
  if ! cmp -s "$f" "$DST/$name"; then
    diffs=$((diffs + 1))
    if [[ $check -eq 1 ]]; then echo "drift: $name differs from SoT"; else cp "$f" "$DST/$name"; echo "synced: $name"; fi
  fi
done
# prune: mirror files with no SoT counterpart are stale
for f in "$DST"/*.json; do
  name="$(basename "$f")"
  if [[ ! -f "$SRC/$name" ]]; then
    diffs=$((diffs + 1))
    if [[ $check -eq 1 ]]; then echo "drift: $name is stale (not in SoT)"; else rm -- "$f"; echo "pruned: $name"; fi
  fi
done

if [[ $check -eq 1 ]]; then
  if [[ $diffs -ne 0 ]]; then
    echo "FAIL: data/agents mirror out of sync ($diffs file(s)) — run 'make sync-agents'" >&2
    exit 1
  fi
  echo "ok: data/agents mirror matches SoT"
else
  echo "ok: synced — data/agents mirror matches SoT ($diffs change(s))"
fi
