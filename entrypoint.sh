#!/bin/sh
# entrypoint.sh — runs as root to fix /data ownership, then drops to the aptify user.
#
# This pattern is necessary because Docker named volumes are created with root
# ownership. The aptify binary runs as a non-root user and needs write access
# to /data for package storage and the SQLite database.
#
# su-exec is a minimal setuid helper (similar to gosu) that replaces the
# current process with the given command running as the target user.
# It is installed in the Dockerfile via: apk add --no-cache su-exec

set -e

# Fix ownership of the data directory if it is owned by root.
if [ "$(stat -c '%u' /data)" = "0" ]; then
    chown -R aptify:aptify /data
fi

# Exec the aptify binary as the aptify user, replacing this shell process.
exec su-exec aptify /app/aptify "$@"
