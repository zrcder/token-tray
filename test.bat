@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   TokenTray 测试模式
echo ========================================
echo.
echo 每 5 秒自动切换场景，共 12 个：
echo   1. 全部正常      2. 渐进上升
echo   3. 接近耗尽      4. 递减
echo   5. 全部危险      6. 周度突高
echo   7. 交错          8. 时度无数据
echo   9. 周度无数据   10. 月度无数据
echo  11. 智谱+DeepSeek 12. API 错误
echo.
echo 右键托盘图标可暂停 / 跳下一个
echo.
TokenTray.exe --test
