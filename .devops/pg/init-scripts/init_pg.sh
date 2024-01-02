#!/usr/bin/env bash

set -euo pipefail

# should be run as superuser
psql -U postgres -f ./init.sql
