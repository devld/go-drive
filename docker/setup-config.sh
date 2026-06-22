#!/bin/sh
# Adjust the bundled config.yml for the Docker image.
#
# Usage: setup-config.sh [config-file]   (defaults to ./config.yml)
#
# - Point the data/web/lang dirs at /app.
# - Enable the thumbnail handlers that need ffmpeg/libvips (installed in the
#   image) by uncommenting the blocks wrapped with docker-handlers markers.
set -e

config="${1:-config.yml}"

sed -i \
    -e 's#data-dir: \./#data-dir: /app/data#' \
    -e 's#web-dir: \./web#web-dir: /app/web#' \
    -e 's#lang-dir: \./lang#lang-dir: /app/lang#' \
    "$config"

sed -i '/docker-handlers:begin/,/docker-handlers:end/{/docker-handlers:/d;s/^    #/    /;}' "$config"
