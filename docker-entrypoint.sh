#!/usr/bin/env bash

set -eou pipefail

exec gosu cataloger /app/cataloger
