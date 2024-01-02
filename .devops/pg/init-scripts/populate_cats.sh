#!/usr/bin/env bash
#
set -euo pipefail

psql -c "\COPY cats(name,age,description) FROM $PWD/cats.csv CSV HEADER;"
