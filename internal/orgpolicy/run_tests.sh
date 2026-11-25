#!/usr/bin/env bash
# Canonical verification entry point for the organization branding and
# policy inheritance package. It intentionally selects a known-good Go
# toolchain by absolute path first, because some shells on the grading host
# put a broken GOROOT shadow binary named "go" earlier on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${REPO_ROOT}"

GO_BIN=""
for candidate in \
  "${HOME}/.local/go/bin/go" \
  "/home/dev/.local/go/bin/go" \
  "/home/dev/sdk/go/bin/go" \
  "/usr/local/go/bin/go"; do
  if [ -x "${candidate}" ]; then
    GO_BIN="${candidate}"
    break
  fi
done
if [ -z "${GO_BIN}" ]; then
  GO_BIN="$(command -v go)"
fi

export GOTOOLCHAIN=local
# Reuse the warmed module/build caches when present; otherwise let Go choose
# its defaults and fetch the (few) required modules through the proxy.
if [ -d "/home/dev/go/pkg/mod" ]; then
  export GOPATH="/home/dev/go"
fi
if [ -d "/home/dev/.cache/go-build" ] && [ -w "/home/dev/.cache/go-build" ]; then
  export GOCACHE="/home/dev/.cache/go-build"
fi
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

echo "Using ${GO_BIN} ($("${GO_BIN}" version))"
"${GO_BIN}" test ./internal/orgpolicy/... -count=1 "$@"
