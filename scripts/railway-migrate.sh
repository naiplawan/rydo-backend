#!/bin/sh
set -eu

exec migrate -path /app/db/migrations -database "${DATABASE_URL:?DATABASE_URL is required}" up
