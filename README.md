# WeekTodo

一个按周记录待办的桌面应用：层级任务、父子联动、跨周带入、周报总结，数据全部保存在本地 JSON 文件里。

![WeekTodo 桌面端](docs/weektodo.png)

使用 Go + Wails v3 + SvelteKit + shadcn-svelte 构建。关闭窗口后常驻 macOS 菜单栏，图标旁显示今天未完成的任务数。

## 功能

- **按周记录**：周一到周日一个任务清单，任务可以标记具体日期，支持「今天 / 未完成 / 已完成 / 已废弃」筛选。
- **层级任务**：任意嵌套的子任务，行首按 Tab 变成上一条的同级子任务，Shift+Tab 提升。
- **父子联动**：子任务全部完成时父任务自动完成；父任务还有未完成子任务时点击复选框会弹确认，确认后一起勾选；取消父任务会连同子任务一起取消。
- **废弃任务**：完成之外还可以废弃，废弃任务不计入完成率和周报，可随时恢复。
- **跨周带入**：一键把上一周未完成的任务复制到本周，保留层级结构，已完成/已废弃的任务不会带入，重复点击不会重复创建。
- **周报**：按完成日期生成精简或完整 Markdown 周报，支持复制和下载；配置任意 OpenAI 兼容接口后可以一键 AI 润色。
- **菜单栏常驻**：关闭窗口后继续在菜单栏运行，图标旁显示今天未完成数量；左键打开窗口，右键快速添加、隐藏或退出。
- **应用内更新**：检查新版本时优先直连 GitHub，失败后自动切换 gh-proxy；下载校验 SHA256，安装 macOS DMG 后自动重启。菜单栏和设置页都可以检查更新。
- **本地优先**：所有数据保存在 `data.json`，每天首次保存自动备份，保留 14 天。

## 运行

需要 Go 1.27.2+、Node.js 22.17+ 和 Wails v3（仅构建桌面应用时需要）。

```bash
# 浏览器 / 开发服务器
make run
# 打开 http://127.0.0.1:8787

# 构建 macOS 应用
make desktop
# 输出 desktop/build/bin/WeekTodo.app
```

开发模式（Vite HMR + Go 后端）：

```bash
make dev
# 前端 http://localhost:5173
# 后端 http://127.0.0.1:8787
```

## 快捷键

| 操作 | 效果 |
|---|---|
| Enter | 在当前任务后新建同级任务 |
| 行首 Tab | 当前任务变成上一条同级任务的子任务 |
| 行首 Shift+Tab | 当前任务提升一级 |
| Alt+↑ / Alt+↓ | 同一层级内上移 / 下移 |
| 空任务行首 Backspace | 删除当前空任务并聚焦上一条 |
| Esc | 退出当前输入 |

## 菜单栏

- 关闭窗口 = 隐藏窗口，应用继续运行。
- 左键点击菜单栏图标打开窗口；右键菜单可以快速添加任务、隐藏窗口或退出。
- 图标旁的数字是今天未完成的叶子任务数量。
- 使用 `Cmd+Q` 或菜单里的「退出 WeekTodo」完全退出。

## 更新

- 菜单栏右键「检查更新…」，或打开「设置 → 应用更新」。
- 检查版本时**优先直接访问 GitHub**；如果 GitHub 请求失败（超时、403、网络不可达等），自动改用 `https://gh-proxy.org/` 代理重试。
- 更新包下载同样优先 GitHub，失败后回退到 gh-proxy。
- 下载完成后用 `SHA256SUMS` / `update.json` 里的 SHA256 校验，再挂载 DMG 替换应用并重启。
- 当前为开发版本（`version=dev`）时不执行更新。

可以通过环境变量覆盖更新源，便于调试：

```bash
WEEKTODO_UPDATE_MANIFEST_URL=...
WEEKTODO_UPDATE_MIRROR=https://gh-proxy.org/
WEEKTODO_UPDATE_API=...
```

## 数据

默认目录（macOS）：`~/Library/Application Support/WeekTodo/`

```text
data.json          所有周的任务
settings.json      AI 配置（权限 0600）
backups/           每天首次保存前的自动备份，保留 14 天
```

`data.json` 是普通 JSON，可以直接备份或版本管理。开发时可以用 `WEEKTODO_DATA_DIR` 覆盖目录。

## AI 周报

在「设置」里填写：

- Base URL，例如 `https://api.deepseek.com/v1`
- 模型，例如 `deepseek-chat`
- API Key

接口需要兼容 `POST {baseUrl}/chat/completions`。API Key 只保存在本机 `settings.json`，不会回传给前端，也不会写进日志。AI 润色失败时会保留本地生成的周报。

## 项目结构

```text
cmd/weektodo/         独立 HTTP 服务入口（开发 / 浏览器）
cmd/update-manifest/  发布时生成 update.json
internal/model/       数据结构
internal/store/       JSON 持久化、原子写入、备份
internal/todo/        周计算、状态推导、跨周带入、周报
internal/ai/          OpenAI 兼容接口
internal/update/      版本检查、GitHub/gh-proxy 回退、下载校验、安装重启
internal/server/      HTTP API + 内嵌前端
frontend/             SvelteKit + shadcn-svelte
desktop/              Wails v3 桌面壳（独立 Go module）
scripts/              开发、打包、图标脚本
```

## 发布

推送 `v*` tag 会触发 `.github/workflows/release.yml`，自动构建服务端二进制、universal macOS DMG、`SHA256SUMS` 和 `update.json`，并发布到 GitHub Releases：

```bash
git tag v0.1.0
git push origin v0.1.0
```

更多开发、测试和发布说明见 [DEVELOP.md](DEVELOP.md)。
