#!/bin/sh
set -eu

DATA_DIR="${DATA_DIR:-/data}"

mkdir -p \
  "${DATA_DIR}/config/source" \
  "${DATA_DIR}/config/runtime" \
  "${DATA_DIR}/state" \
  "${DATA_DIR}/ui"

exec /usr/local/bin/gatewayd
