#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 build <tag> <version> <commit> | prove <image> <version>" >&2
  exit 2
}

build() {
  [ "$#" -eq 3 ] || usage
  local tag="$1" version="$2" commit="$3" iidfile
  iidfile="$(mktemp)"
  docker build --iidfile "$iidfile" \
    --build-arg "VERSION=$version" \
    --build-arg "COMMIT=$commit" \
    --label "org.opencontainers.image.revision=$commit" \
    --label "org.opencontainers.image.version=$version" \
    -t "$tag" . >&2
  cat "$iidfile"
  rm -f "$iidfile"
}

container=""
cleanup() {
  [ -z "$container" ] || docker rm -f "$container" >/dev/null 2>&1 || true
}

prove() {
  [ "$#" -eq 2 ] || usage
  local image="$1" version="$2" port="${PROOF_PORT:-7100}" code out
  out="$(docker run --rm "$image" version)"
  case "$out" in
    "chill-mcp $version ("*) ;;
    *) echo "expected version $version, got: $out" >&2; exit 1 ;;
  esac
  container="chill-mcp-proof-$$"
  trap cleanup EXIT
  docker run -d --name "$container" --read-only --cap-drop ALL --security-opt no-new-privileges:true \
    -e CHILL_ENGINE_BASE_URL=http://127.0.0.1:9 -p "127.0.0.1:$port:7100" "$image" >/dev/null
  for _ in $(seq 1 50); do docker exec "$container" /chill-mcp health 2>/dev/null && break; sleep 0.2; done
  docker exec "$container" /chill-mcp health
  code="$(curl -s -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:$port/" -H 'Content-Type: application/json' -d '{}')"
  [ "$code" = "401" ] || { echo "expected 401 without bearer, got $code" >&2; exit 1; }
  printf 'proved %s\n' "$(docker image inspect "$image" --format '{{.Id}}')"
}

[ "$#" -ge 1 ] || usage
cmd="$1"
shift
case "$cmd" in
  build) build "$@" ;;
  prove) prove "$@" ;;
  *) usage ;;
esac
