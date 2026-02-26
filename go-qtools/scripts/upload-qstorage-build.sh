#!/usr/bin/env bash
set -euo pipefail

DEFAULT_REGION="q-world-1"
DEFAULT_ENDPOINT_URL="https://qstorage.quilibrium.com"
DEFAULT_SOURCE_DIR="dist"
DEFAULT_PREFIX="qtools/dev-builds"

usage() {
  cat <<'EOF'
Upload qtools build artifacts to QStorage (S3-compatible) with rotation.

Behavior:
1) Moves existing objects from <prefix>/current/ to <prefix>/old/<timestamp>/
2) Uploads latest local binaries to <prefix>/current/

Usage:
  scripts/upload-qstorage-build.sh [options]

Options:
  --bucket <name>             S3/QStorage bucket name (required; prompts if missing)
  --access-key-id <id>        Access key ID (required; prompts if missing)
  --access-key <key>          Access key / secret (required; prompts if missing)
  --account-id <id>           Optional account ID (accepted for QStorage parity)
  --source-dir <path>         Local directory containing binaries (default: dist)
  --prefix <path>             Bucket prefix root (default: qtools/dev-builds)
  --region <region>           S3 region (default: q-world-1)
  --endpoint-url <url>        S3 endpoint URL (default: https://qstorage.quilibrium.com)
  --artifacts <csv>           Comma-separated artifact filenames (default: qtools,qtools-arm64)
  --dry-run                   Print planned operations without uploading
  -h, --help                  Show this help
EOF
}

trim_trailing_slash() {
  local value="$1"
  while [[ "$value" == */ ]]; do
    value="${value%/}"
  done
  printf '%s' "$value"
}

S3_BUCKET=""
ACCESS_KEY_ID=""
ACCESS_KEY=""
ACCOUNT_ID=""
SOURCE_DIR="$DEFAULT_SOURCE_DIR"
PREFIX="$DEFAULT_PREFIX"
REGION="$DEFAULT_REGION"
ENDPOINT_URL="$DEFAULT_ENDPOINT_URL"
ARTIFACTS_CSV="qtools,qtools-arm64"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bucket)
      S3_BUCKET="${2:-}"
      shift 2
      ;;
    --access-key-id)
      ACCESS_KEY_ID="${2:-}"
      shift 2
      ;;
    --access-key)
      ACCESS_KEY="${2:-}"
      shift 2
      ;;
    --account-id)
      ACCOUNT_ID="${2:-}"
      shift 2
      ;;
    --source-dir)
      SOURCE_DIR="${2:-}"
      shift 2
      ;;
    --prefix)
      PREFIX="${2:-}"
      shift 2
      ;;
    --region)
      REGION="${2:-}"
      shift 2
      ;;
    --endpoint-url)
      ENDPOINT_URL="${2:-}"
      shift 2
      ;;
    --artifacts)
      ARTIFACTS_CSV="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if ! command -v aws >/dev/null 2>&1; then
  echo "Error: aws CLI is required but not found in PATH." >&2
  exit 1
fi

if [[ -z "$S3_BUCKET" ]]; then
  read -r -p "QStorage bucket name: " S3_BUCKET
fi
if [[ -z "$ACCESS_KEY_ID" ]]; then
  read -r -p "QStorage access key ID: " ACCESS_KEY_ID
fi
if [[ -z "$ACCESS_KEY" ]]; then
  read -r -s -p "QStorage access key: " ACCESS_KEY
  echo
fi

if [[ -z "$S3_BUCKET" || -z "$ACCESS_KEY_ID" || -z "$ACCESS_KEY" ]]; then
  echo "Error: bucket, access key ID, and access key are required." >&2
  exit 1
fi

if [[ -n "$ACCOUNT_ID" ]]; then
  echo "Info: account ID provided and accepted for QStorage compatibility."
fi

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "Error: source directory not found: $SOURCE_DIR" >&2
  exit 1
fi

IFS=',' read -r -a ARTIFACTS <<<"$ARTIFACTS_CSV"
if [[ "${#ARTIFACTS[@]}" -eq 0 ]]; then
  echo "Error: at least one artifact must be specified." >&2
  exit 1
fi

MISSING=()
for artifact in "${ARTIFACTS[@]}"; do
  artifact="$(echo "$artifact" | xargs)"
  if [[ -z "$artifact" ]]; then
    continue
  fi
  if [[ ! -f "$SOURCE_DIR/$artifact" ]]; then
    MISSING+=("$artifact")
  fi
done

if [[ "${#MISSING[@]}" -gt 0 ]]; then
  echo "Error: missing artifact(s) in $SOURCE_DIR: ${MISSING[*]}" >&2
  exit 1
fi

PREFIX="$(trim_trailing_slash "$PREFIX")"
BASE_URI="s3://$S3_BUCKET"
if [[ -n "$PREFIX" ]]; then
  BASE_URI="$BASE_URI/$PREFIX"
fi

CURRENT_URI="$BASE_URI/current/"
ROTATION_TS="$(date -u +%Y%m%dT%H%M%SZ)"
OLD_URI="$BASE_URI/old/$ROTATION_TS/"

AWS_BASE_ARGS=(--endpoint-url "$ENDPOINT_URL" --region "$REGION")
if [[ "$DRY_RUN" == true ]]; then
  AWS_BASE_ARGS+=(--dryrun)
fi

aws_s3() {
  AWS_ACCESS_KEY_ID="$ACCESS_KEY_ID" \
  AWS_SECRET_ACCESS_KEY="$ACCESS_KEY" \
  AWS_EC2_METADATA_DISABLED=true \
  aws "${AWS_BASE_ARGS[@]}" "$@"
}

echo "Uploading qtools artifacts to QStorage"
echo "  Bucket:          $S3_BUCKET"
echo "  Prefix:          ${PREFIX:-<root>}"
echo "  Region:          $REGION"
echo "  Endpoint:        $ENDPOINT_URL"
echo "  Source directory: $SOURCE_DIR"
echo "  Current path:    $CURRENT_URI"
echo "  Old path:        $OLD_URI"
echo "  Dry run:         $DRY_RUN"

CURRENT_LIST="$(aws_s3 s3 ls "$CURRENT_URI" --recursive 2>/dev/null || true)"
if [[ -n "$CURRENT_LIST" ]]; then
  echo "Rotating existing current artifacts to old/$ROTATION_TS ..."
  aws_s3 s3 cp "$CURRENT_URI" "$OLD_URI" --recursive --only-show-errors
  aws_s3 s3 rm "$CURRENT_URI" --recursive --only-show-errors
else
  echo "No existing artifacts under current/ to rotate."
fi

echo "Uploading latest artifacts to current/ ..."
for artifact in "${ARTIFACTS[@]}"; do
  artifact="$(echo "$artifact" | xargs)"
  [[ -z "$artifact" ]] && continue
  local_path="$SOURCE_DIR/$artifact"
  remote_path="$CURRENT_URI$artifact"
  echo "  -> $artifact"
  aws_s3 s3 cp "$local_path" "$remote_path" --only-show-errors
done

echo "Upload complete."
