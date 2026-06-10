#!/usr/bin/env sh
set -eu

PACKAGE="${PRISMGO_INSTALLER_PACKAGE:-github.com/prismgo/installer/cmd/prismgo@latest}"

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is required to install the PrismGo installer." >&2
  echo "Install Go first, then rerun this script." >&2
  exit 1
fi

echo "Installing prismgo from ${PACKAGE}..."
go install "${PACKAGE}"

gobin="$(go env GOBIN 2>/dev/null || true)"
if [ -z "${gobin}" ]; then
  gopath="$(go env GOPATH 2>/dev/null || true)"
  gobin="${gopath}/bin"
fi

echo "prismgo installed."
echo "Ensure this directory is on your PATH:"
echo "  ${gobin}"
