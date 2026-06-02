# 改动记录

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
