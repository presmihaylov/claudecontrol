#!/bin/bash

set -e

if [ -z "$DB_URL" ]; then
    echo "Error: DB_URL environment variable is not set"
    exit 1
fi

MIGRATIONS_DIR="/tmp/migrations"

echo "Cleaning up migrations directory..."
rm -rf "$MIGRATIONS_DIR"
mkdir -p "$MIGRATIONS_DIR"

echo "Copying and renaming migration files..."
for file in supabase/migrations/*.sql; do
    if [ -f "$file" ]; then
        basename=$(basename "$file" .sql)
        cp "$file" "$MIGRATIONS_DIR/${basename}.up.sql"
        echo "Copied: $file -> $MIGRATIONS_DIR/${basename}.up.sql"
    fi
done

echo "Running migrations..."
migrate -source "file://$MIGRATIONS_DIR/" -database "$DB_URL" up

echo "Database migration completed successfully!"