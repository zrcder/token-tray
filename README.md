# TokenTray

> macOS 菜单栏大模型 API 用量监控 — 智谱 GLM / DeepSeek

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![macOS](https://img.shields.io/badge/macOS-13.0+-000000?logo=apple)](https://www.apple.com/macos/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

TokenTray 在 macOS 菜单栏实时显示大模型 API 的额度用量。不用打开浏览器，瞄一眼就知道还剩多少。

## 菜单栏图标

图标是一个 **3 段彩色条**，每段代表一个额度窗口，按用量严重程度排序取最高的 3 个显示：

| 图标状态 | 含义 |
|---------|------|
| 🟩🟩🟩 三段全绿 | 所有额度用量 < 70%，安全 |
| 🟨🟩🟩 一段黄 | 有一个额度窗口达到 70%-90%，注意 |
| 🟥🟩🟩 一段红 | 有一个额度窗口 > 90%，危险 |
| 🟥🟥🟩 两段红 | 多个额度窗口即将耗尽 |

颜色规则：
- **绿色** ■ — 用量 < 70%
- **黄色** ■ — 用量 70% - 90%
- **红色** ■ — 用量 > 90%

鼠标悬停显示各窗口的百分比摘要。点击展开完整下拉面板。

## 下拉面板

```
🟢 智谱 GLM · MAX
   时度  ■□□□□□□□□□  10%
   周度  ■■■□□□□□□□  26%  5d8h
   月度  □□□□□□□□□□   0%

🟢 DeepSeek · ¥275.92
   余额  □□□□□□□□□□  10%

⏱ 14:32
─────────────────────
↻ 刷新
─────────────────────
⚙ 智谱 Key         ●
⚙ DeepSeek Key     ●
⏱ 每 5 分钟刷新
─────────────────────
退出
```

- **🟢/🟡/🔴** — Provider 整体状态
- **时度/周度/月度** — 智谱的 5 小时 / 每周 / MCP 月度窗口
- **■□** — 进度条（黑色方块=已用，白色方块=剩余）
- **百分比** — 右对齐，个位数补空格
- **倒计时** — 窗口重置剩余时间

## 功能

- 📊 **多 Provider 监控** — 智谱 GLM + DeepSeek，同一面板展示
- 🎨 **3 段彩色图标** — 用量最高的 3 个窗口，颜色直观看状态
- ⏱ **可配置刷新间隔** — 1 / 5 / 10 / 30 分钟，菜单里点击切换
- ⚙️ **应用内设置** — 原生 macOS 对话框配置 API Key，无需编辑文件
- 📦 **原生体验** — Go + AppKit，5.7MB 二进制，零运行时依赖

## 安装

### 下载 .dmg（推荐）

1. 从 [Releases](../../releases) 下载 `TokenTray.dmg`
2. 双击打开，拖 TokenTray.app 到 Applications
3. 启动后点击菜单栏图标 → **⚙ 智谱 Key** → 粘贴 API Key → 保存

### 从源码构建

```bash
git clone https://github.com/qs2d/token-tray.git
cd token-tray
./run.sh rebuild
```

前提：Go 1.21+、macOS Command Line Tools、macOS 13.0+

## 配置

### 智谱 GLM

1. 获取 API Key：https://open.bigmodel.cn/usercenter/apikeys
2. 菜单栏 → 点击图标 → **⚙ 智谱 Key** → 粘贴 → 保存

API Key 格式：`xxxxxxxx.yyyyyyyyyyyy`（整体一串，含中间的点）

### DeepSeek

1. 获取 API Key：https://platform.deepseek.com/api_keys
2. 菜单栏 → 点击图标 → **⚙ DeepSeek Key** → 粘贴（sk- 开头）→ 保存

## 常见问题

**菜单栏看不到图标？**
- 确认 macOS ≥ 13.0
- 如果装了 Bartender / Ice，在管理 App 中把 TokenTray 设为常驻显示
- macOS 15.3 存在 WindowServer 渲染 bug，升级到 15.7+ 可解决

**显示 API Key 无效？**
- 智谱：确认 Key 含中间的点，从 [API Keys 页面](https://open.bigmodel.cn/usercenter/apikeys) 获取
- DeepSeek：确认 sk- 开头

## 技术架构

```
token-tray/
├── main.go        入口
├── app.go         菜单栏 UI + 轮询 + 设置弹窗
├── provider.go    Provider 接口 + 数据模型
├── zhipu.go       智谱 GLM provider
├── deepseek.go    DeepSeek provider
├── config.go      JSON 配置持久化（原子写入）
├── format.go      进度条/数字/时间格式化
├── icon.go        PNG 图标运行时生成
├── build.sh       编译 + .app bundle
└── run.sh         一键启动
```

### Provider 接口

```go
type Provider interface {
    Name() string
    ShortLabel() string
    FetchStatus() (*UsageReport, error)
}
```

添加新供应商：创建 `xxx.go` 实现接口，在 `app.go` 注册。UI 自动适配。

### 智谱 API

- 端点：`GET https://open.bigmodel.cn/api/monitor/usage/quota/limit`
- 认证：`Authorization: <API_KEY>`（无 Bearer 前缀）
- Coding Plan 返回两个 TOKENS_LIMIT（5 小时窗口 + 每周窗口）

### DeepSeek API

- 端点：`GET https://api.deepseek.com/user/balance`
- 认证：`Authorization: Bearer <API_KEY>`
- 返回剩余余额（¥），按阈值映射为颜色状态

## 路线图

- [x] 智谱 GLM Coding Plan 用量监控
- [x] DeepSeek 余额监控
- [x] 3 段彩色图标
- [x] 可配置刷新间隔
- [x] 应用内设置弹窗
- [x] .dmg 安装包
- [ ] 用量超阈值系统通知
- [ ] 更多 Provider（OpenAI / Anthropic）

## License

[MIT](LICENSE)
