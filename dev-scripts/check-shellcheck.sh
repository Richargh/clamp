#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."
shellcheck ./*.sh ./dev-scripts/*.sh ./startup-scripts/*.sh ./shutdown-scripts/*.sh
