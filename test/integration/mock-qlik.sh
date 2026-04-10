#!/bin/sh
# test/integration/mock-qlik.sh
# Resolve real script location even when invoked through a symlink.
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")")" && pwd)"
case "$*" in
  "space ls --json")
    cat "$SCRIPT_DIR/testdata/spaces.json"
    ;;
  "app ls --json --limit 1000")
    cat "$SCRIPT_DIR/testdata/apps.json"
    ;;
  "app unbuild --app"*)
    DIR=$(echo "$*" | sed 's/.*--dir //')
    mkdir -p "$DIR"
    echo "resourceId: test" > "$DIR/config.yml"
    echo "LOAD * FROM test.qvd;" > "$DIR/script.qvs"
    echo "[]" > "$DIR/measures.json"
    echo "[]" > "$DIR/dimensions.json"
    echo "[]" > "$DIR/variables.json"
    ;;
  "context ls")
    echo "default *"
    ;;
  *)
    echo "mock: unknown command: $*" >&2
    exit 1
    ;;
esac
