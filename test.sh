#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "═══════════════════════════════════════"
echo "  TokenTray 自诊断测试"
echo "═══════════════════════════════════════"
echo ""

# 1. Generate test icon PNGs
echo "[1/3] 生成测试图标 PNG..."
CGO_ENABLED=1 go build -o TokenTray . 2>/dev/null
./TokenTray --gen-icons
echo "  ✅ 已生成 $(ls screenshots/icon-test-*.png | wc -l | tr -d ' ') 个测试图标"
echo ""

# 2. Show icon inventory
echo "[2/3] 图标清单:"
for f in screenshots/icon-test-*.png; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f" 2>/dev/null)
  printf "  %-35s %s bytes\n" "$f" "$size"
done
echo ""

# 3. Launch test mode
echo "[3/3] 启动测试模式（虚拟数据，无 API 调用）..."
echo ""
echo "  菜单栏将显示 3 段: 橙色 + 红色 + 红色"
echo "  下拉菜单包含:"
echo "    智谱 GLM (测试)  — 时度 18% / 周度 42% / 月度 68%"
echo "    DeepSeek (测试)  — 余额 88%"
echo "    边缘场景 (测试)  — 时度 —  / 周度 95%"
echo ""
echo "  按 Ctrl+C 退出"
echo ""

if [[ "$OSTYPE" == "darwin"* ]]; then
  CGO_ENABLED=1 go build -ldflags="-s -w" -o TokenTray . 2>/dev/null
  ./TokenTray --test
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
  ./TokenTray.exe --test
else
  ./TokenTray --test
fi
