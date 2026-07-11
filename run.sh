#!/bin/bash
# Launch TokenTray as a menu bar app.
#   ./run.sh            # build if needed, then open
#   ./run.sh rebuild     # force rebuild
#   ./run.sh debug       # foreground binary (logs to terminal)
set -euo pipefail
cd "$(dirname "$0")"

APP="dist/TokenTray.app"
MODE="${1:-launch}"

if [ "$MODE" = "debug" ]; then
    echo "[debug] foreground binary..."
    exec ./TokenTray
fi

if [ ! -d "$APP" ] || [ "$MODE" = "rebuild" ]; then
    ./build.sh
fi

echo "[launch] open $APP"
echo "[launch] 看菜单栏右上角 → 应出现「智 X%」"
open "$APP"
