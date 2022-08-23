#!/bin/sh
set -e

LOG_DIR=/tmp/go-drive-thumbnail.log

echo "Type: $GO_DRIVE_ENTRY_TYPE" >> $LOG_DIR
echo "RealPath: $GO_DRIVE_ENTRY_REAL_PATH" >> $LOG_DIR
echo "Path: $GO_DRIVE_ENTRY_PATH" >> $LOG_DIR
echo "Type: $GO_DRIVE_ENTRY_TYPE" >> $LOG_DIR
echo "Name $GO_DRIVE_ENTRY_NAME" >> $LOG_DIR
echo "Size: $GO_DRIVE_ENTRY_SIZE" >> $LOG_DIR
echo "ModTime: $GO_DRIVE_ENTRY_MOD_TIME" >> $LOG_DIR
echo "URL: $GO_DRIVE_ENTRY_URL" >> $LOG_DIR
echo >> $LOG_DIR

# ffmpeg read from stdin and write to stdout
ffmpeg -hide_banner -loglevel error -i - -frames:v 1 -vf scale=220:-1 -f mjpeg -

# or

# ffmpeg read from url and write to stdout
#ffmpeg -hide_banner -loglevel error -i http://localhost:8089$GO_DRIVE_ENTRY_URL -frames:v 1 -vf scale=220:-1 -f mjpeg -
