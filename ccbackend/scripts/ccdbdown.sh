#!/bin/bash

set -e

if [ -z "$DB_URL" ]; then
    echo "Error: DB_URL environment variable is not set"
    exit 1
fi

echo "Dropping claudecontrol schemas and migration table..."

psql "$DB_URL" -c "DROP SCHEMA IF EXISTS claudecontrol CASCADE;"
psql "$DB_URL" -c "DROP SCHEMA IF EXISTS claudecontrol_test CASCADE;"
psql "$DB_URL" -c "DROP TABLE IF EXISTS public.schema_migrations;"

echo "Schemas and migration table dropped successfully!"