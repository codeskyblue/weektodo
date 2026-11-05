# WeekTodo Development

构建、测试和发布说明。安装与使用见 [README.md](README.md)。

## 环境

- Go 1.27.2+
- Node.js 22.17+
- Wails v3.0.0（仅构建桌面应用）
- macOS 构建图标需要 `sips` 和 `wails3`

## 常用命令

```bash
make frontend   # npm install + SvelteKit 静态构建
make build      # 构建 bin/weektodo
make dev        # Vite HMR + Go 文件监听
make run        # make build 后启动 HTTP 服务
make test       # 构建前端后运行 go test ./...
make check      # svelte-check
make desktop    # 构建 universal macOS .app
make icons      # 从 SVG 重新生成托盘/应用图标
```

开发模式：

```bash
make dev
# 前端 http://localhost:5173
# 后端 http://127.0.0.1:8787
# 数据目录 ~/.config/weektodo/
```

可用环境变量：

```bash
DEV_BACKEND_PORT=8788 DEV_FRONTEND_PORT=5174 make dev
WEEKTODO_DATA_DIR=/tmp/weektodo-dev make dev
```

## 架构

前端是 SvelteKit 静态站点，构建到 `frontend/build`，通过 `//go:embed all:build`
嵌入 Go 二进制。`internal/server` 同时提供 `/api/*` 和静态文件，Wails 桌面壳的
`Assets.Handler` 复用同一个 handler，所以浏览器和桌面端行为一致。

- `internal/model`：`Week` / `Task` / `Settings`
- `internal/store`：读取、原子写入、每日备份、今日计数、跨周移动
- `internal/todo`：ISO 周计算、父状态推导、跨周带入、跨周移动、周报
- `internal/ai`：OpenAI 兼容的 `/chat/completions`
- `internal/server`：REST API、设置、静态资源
- `desktop`：Wails v3 窗口、菜单栏托盘、今日数量

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查 |
| GET | `/api/meta` | 数据目录、版本、当前周 |
| GET | `/api/weeks` | 周列表 |
| GET | `/api/weeks/{id}` | 读取一周，`?optional=1` 时不存在返回 `week: null` |
| POST | `/api/weeks/{id}` | 创建空周 |
| PUT | `/api/weeks/{id}` | 保存整周任务树 |
| POST | `/api/weeks/{id}/move` | 把任务子树移到另一周 |
| DELETE | `/api/weeks/{id}` | 删除一周 |
| GET | `/api/weeks/{id}/carry-over` | 预览上周可带入任务 |
| POST | `/api/weeks/{id}/carry-over` | 执行带入 |
| GET | `/api/weeks/{id}/report?mode=brief\|full` | 本地周报 |
| POST | `/api/weeks/{id}/report/polish` | AI 润色 |
| GET/PUT | `/api/settings` | AI 设置（Key 只写不读） |
| POST | `/api/settings/ai/test` | 测试 AI 连接 |
| POST | `/api/settings/ai/models` | 获取 OpenAI 兼容模型列表 |
| GET | `/api/update/check` | 检查最新版本 |
| POST | `/api/update/apply` | 下载并安装更新 |
| GET | `/api/update/status` | 下载/安装进度 |

周 ID 使用 ISO 周格式，例如 `2026-W45`。每天首次保存前会把当前 `data.json`
复制到 `backups/data-YYYY-MM-DD.json`，只保留最近 14 份。

## 数据与状态规则

- 任务以嵌套 `children` 保存，最大深度 8 层，单周最多 2000 个任务。
- 父任务状态由非废弃子任务推导：全部完成 → 完成；否则 → 未完成。
- 废弃父任务会级联废弃整棵子树；恢复时整棵子树恢复为未完成。
- 完成率与周报只统计叶子任务，已废弃任务不计入分母。
- 跨周带入只复制 `todo` 节点，已完成/已废弃节点跳过，`day` 清空，并记录
  `carriedFrom` 保证幂等。

## 测试

```bash
make test
```

Go 测试覆盖：

- ISO 周计算与跨年边界
- 状态推导、深度限制、日期清理、废弃级联
- 统计、今日计数、跨周带入幂等
- 周报精简/完整模式
- JSON 读写、0600 权限、每日备份、损坏恢复
- OpenAI 兼容请求、错误处理、提示词
- HTTP API、设置脱敏、静态资源

前端：

```bash
cd frontend
npm run check
npm run build
```

界面流程用 Playwright / playwright-cli 对临时后端验证：新增任务、行首 Tab 缩进、
父任务确认完成、废弃/恢复、筛选、跨周带入、周报、设置和刷新持久化。

## 应用更新

`internal/update` 同时支持服务端二进制和 macOS 桌面应用：

1. **检查版本**：优先直连 GitHub（`releases/latest/download/update.json`、GitHub latest 重定向、GitHub API 依次尝试）；直连失败后，使用 `https://gh-proxy.org/` 代理同样的 GitHub URL 重试。
2. **下载更新**：同样先走 GitHub release asset，失败后回退到 gh-proxy。
3. **校验**：优先使用 manifest 里的 SHA256，否则下载 `SHA256SUMS` 校验。
4. **安装**：桌面端挂载 DMG，替换 `WeekTodo.app` 内的二进制和 `Info.plist`，重新 ad-hoc 签名；服务端替换自身二进制。
5. **重启**：`syscall.Exec` 重新启动当前进程。

更新源可用 `WEEKTODO_UPDATE_*` 环境变量覆盖。默认镜像为 `https://gh-proxy.org/`。

## 发布

推送 `v*` tag 会触发 `.github/workflows/release.yml`：

- 构建前端
- 构建 `weektodo-server-<platform>-<arch>-<version>` 服务端二进制
- 构建 universal macOS `WeekTodo.app` 并打包为 `WeekTodo-desktop-mac-<version>.dmg`
- 生成 `SHA256SUMS` 和 `update.json`
- 上传 GitHub Release

Release Notes 由 `scripts/release-notes.sh` 从上一个 tag 到当前 tag 的提交自动生成，按 `新增 / 修复 / 重构 / 其他` 分类，并作为 Release 正文；`cmd/update-manifest` 再从 `SHA256SUMS`、这份正文和发布时间生成 `update.json`，所以应用内显示的更新内容和 Release 页面保持一致。

发布前可以本地预览：

```bash
./scripts/release-notes.sh v0.2.0
```

## 多周视图

- `settings.json` 的 `weekSpan` 保存 1 / 2 / 3 周，默认 1。
- 前端以当前浏览周为锚点计算窗口：2 周为「上一周 + 当前」，3 周为「上一周 + 当前 + 下一周」。左右箭头平移锚点。
- 相邻周通过 `GET /api/weeks/{id}?optional=1` 读取；不存在时前端生成内存占位，只有添加任务或移入任务时才写入 `data.json`。
- `POST /api/weeks/{id}/move` 调用 `Store.MoveTask`：克隆源周和目标周、摘下整棵子树、清空日期和 `carriedFrom`、更新时间戳，最后一次性保存。目标周不存在时自动创建。
- 前端按周维护脏标记，统一防抖保存；切换窗口、跨周移动或 WebDAV 同步前会先等待未完成的保存。
- 多周视图每列复用 `TaskList` 和紧凑版 `QuickAdd`，`TaskRow` / `DayPicker` 通过任务 ID 定位所属周。

## WebDAV 同步

- `internal/webdav`：Basic Auth 客户端，负责 `PROPFIND` 测试、`MKCOL` 建目录、`GET` / `PUT` 文件。
- `internal/todo/merge.go`：按任务 ID 和更新时间合并两份 JSON，子任务取并集。
- `POST /api/webdav/test`：用草稿配置（空密码沿用已保存密码）测试连接。
- `POST /api/webdav/sync`：`Snapshot` 本地数据 → 下载远端 `weektodo.json` → `MergeData` → `ReplaceData` → 上传合并结果；远端 JSON 解析失败时不修改任何数据。
- 配置保存在 `settings.json` 的 `webdav` 字段，密码不会通过 `GET /api/settings` 返回，只返回末四位提示。
- `remoteDir` 保存时统一加前导 `/`（如 `/weektodo`）用于展示，实际仍相对于 WebDAV 地址拼接。
- 同步失败会保留本地数据；上传失败时本地已写入合并结果，可以再次同步重试。

## 桌面构建

```bash
make frontend
make desktop
# desktop/build/bin/WeekTodo.app
```

构建脚本分别编译 `amd64` 和 `arm64`，用 `lipo` 合并，复制 `Info.plist` 和
`AppIcon.icns`，最后 ad-hoc 签名。应用打开窗口时使用常规激活策略，出现在
Dock、Cmd+Tab 和菜单栏；关闭窗口后切换为 Accessory，从 Dock/Cmd+Tab 移除，
仅保留菜单栏图标。点击菜单栏图标会恢复 Regular 模式。

托盘图标和菜单栏数字由 `desktop/main.go` 维护：

- `tray.SetTemplateIcon` 显示单色模板图标
- `tray.SetLabel` 显示今天未完成叶子任务数
- 前端保存成功后发送 `weektodo:changed` 事件，桌面进程刷新数量
- 另每 2 秒轮询本地 store 兜底刷新（事件可能丢失）
- 关闭窗口时发送 `weektodo:window-hiding`，前端先 flush 未保存内容
