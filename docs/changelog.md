# 改动记录

## [v0.0.4] - 2026-06-06

### ACP 协议完整集成

**文件**: `agent/`, `client/`, `cmd/bubblecode/acp.go`, `cmd/bubblecode/chat.go`

用真实 LLM Agent（OpenAI 兼容 API）替换 mock agent。架构改为 `chat` 命令启动 TUI → 子进程 `bubblecode acp` 运行 ACP server → stdin/stdout 管道通信。新增 `PromptRunner` 管理会话生命周期，支持 `SessionUpdate` 消息驱动渲染。

### TUI: 会话管理 + 面板系统

**文件**: `tui/`, `cmd/bubblecode/chat.go`

- 新增 session 面板（Ctrl+S / `/sessions`）：创建、切换会话，`/new` 创建新会话
- 新增 models 面板（Ctrl+M / `/models`）：切换当前模型
- 新增 commands 面板（Ctrl+P）：命令列表，简化至 3 项（New Session / Switch Session / Change Model）
- slash 建议行内渲染：输入 `/` 时在 textarea 上方显示匹配命令
- Focus 类型重构：用 `FocusChat`/`FocusCommands`/`FocusSessions`/`FocusModels` 替代 bool
- 组件文件重命名为 snake_case

### 配置系统改造

**文件**: `agent/config.go`, `cmd/bubblecode/providers.go`, `cmd/bubblecode/models.go`

- 多 provider 支持，`~/.config/bubblecode/config.json`
- `BUBBLECODE_API_KEY` 环境变量作为 fallback
- 使用 `huh` 替代 `bufio.Reader` 做交互式配置
- 旧单 key 配置自动迁移
- provider/models 子命令：添加、删除、选择默认 provider/model

### CI 与构建

**文件**: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `Makefile`

- `make fmt` 退出码 1（之前静默接受不格式化代码）
- `make build` 仅构建当前平台二进制，取消全平台交叉编译
- CI 触发分支从 `main` 改为 `master`

### 修复

- 命令行 `Enter` 发送 slash 命令不匹配时回退为发送提示
- scrollbar 高度与 thumb 比例计算（使用 `TotalLineCount`）
- viewport 初始高度使用 `GetChatHeight` 而非 `InitHeight`
- commands overlay 布局改用 bubbleflex，Width 修正为 52
- config 文件权限设为 0600，路径遍历 sandbox 保护
- 删除 `opencode` CLI 依赖，改为 spawn 自身子进程

## [2026-05-23]

### Ctrl+C 退出程序

**文件**: `tui/update.go`

**问题**: Ctrl+C 无法退出程序。

**根因**: bubbletea v2 的 `KeyMsg` 接口由 `KeyPressMsg`/`KeyReleaseMsg` 实现。原来的 `KeyPressMsg` 分支用 `Mod+Code` 硬匹配 Ctrl+C，之后再无冗余的 `KeyMsg` 分支（死代码）。但当 `showCommands=true` 时，`handleKey` 在 `showCommands` 块中只响应 `esc`/`ctrl+p`，其他键全部忽略。如果某些 Windows 终端以 `KeyMsg` 接口而非 `KeyPressMsg` 投递，就会绕过 `KeyPressMsg` 分支、走进 `handleKey` 被 `showCommands` 吞掉。

**修复**:
- 移除 `Update` 中 `tea.KeyPressMsg` 的 `Mod+Code` 匹配
- 移除 `tea.KeyMsg` 死代码分支
- 所有键（`KeyPressMsg`）统一走 `handleKey`
- `ctrl+c` 移到 `handleKey` 最顶部，在 `showCommands` 检查之前，确保任何状态下都能退出

### Windows 终端 resize 后 viewport 不更新

**文件**: `tui/model.go`, `tui/update.go`

**问题**: 调整终端窗口大小后，聊天 viewport 的宽高没有同步更新。

**根因**: bubbletea v2 在 Windows 上没有 `SIGWINCH` 支持（`signals_windows.go:listenForResize` 为空实现）。`WindowSizeMsg` 只在启动时发送一次，后续 resize 永不通知 model。viewport 的初始宽高（`InitWidth=100`）永远不变。

**修复**:
- 添加 `resizePollMsg` + `pollResize()`，每 500ms 通过 `tea.RequestWindowSize` 查询终端尺寸
- `tea.RequestWindowSize` → 框架调用 `checkResize()` → `term.GetSize()` 获取实际终端尺寸 → 发送 `WindowSizeMsg`
- `handleResize` 添加 guard：尺寸未变时直接 `return m, nil`，避免无效重渲染

### 透明遮罩支持（CompositeMasked）

**文件**: `tui/overlay/overlay.go`, `docs/overlay.md`

详见 `docs/overlay.md`。
