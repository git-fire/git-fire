#!/usr/bin/env bash
# Convenience alias for the MVP UAT suite (see scripts/uat_test.sh).
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/uat_test.sh" "$@"
