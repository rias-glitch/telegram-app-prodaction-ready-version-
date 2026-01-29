#!/bin/sh
set -e

echo "Applying migrations..."
/bin/migrate_apply -apply

echo "Starting app"
/bin/app
