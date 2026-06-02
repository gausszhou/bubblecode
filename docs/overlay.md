# Overlay 包 — 透明遮罩实现原理

## 概述

`tui/overlay` 提供 ANSI 终端字符串的层叠合成能力，支持全屏透明遮罩效果。

## Composite — 基础合成

将前景字符串（fg）按指定位置叠到背景字符串（bg）上：

```
fg  "┌───┐"
    "│ x │"
    "└───┘"

bg  "AAAAAAAAA"
    "BBBBBBBBB"
    "CCCCCCCCC"

↓ Composite(fg, bg, Center, Center, 0, 0)

    "AAAAAAAAA"
    "BBB┌───┐C"
    "CCC│ x │C"
    "CCC└───┘C"
```

规则：
- fg 完全覆盖 bg 在合成区域的每个 cell
- 合成区域之外的 bg 保持不变
- ANSI 样式被保留

## CompositeMasked — 透明遮罩合成

在 `Composite` 之上增加全屏半透明遮罩：

```go
CompositeMasked(fg, bg, Center, Center, 0, 0)         // 默认启用遮罩
CompositeMasked(fg, bg, Center, Center, 0, 0, false)   // 关闭遮罩
```

**合成流程**（两步）：

```
1. applyScrim(bg)
   对 bg 每一行手工构造 ANSI escape codes：
   - 行首：\x1b[2m (dim on) + \x1b[48;2;R;G;Bm (dark bg)
   - 行内：在每个 ANSI reset (\x1b[0m, \x1b[m, \x1b[0;*m) 后
           重新注入 \x1b[2m + \x1b[48;2;R;G;Bm
   - 行尾：\x1b[22m (dim off) + \x1b[49m (bg default)

2. Composite(fg, scrimmedBg, ...)
   前景叠在遮罩后的背景上
```

**为什么不用 lipgloss.Style.Render？**

`Style.Background(dark).Faint(true).Render(line)` 会在整行前后加 dim on/off，
但行内若包含 `\x1b[0m`（ANSI reset），会清除 dim 和背景色。
手工构造 ANSI 并在每个 reset 后重新注入，确保遮罩效果不受内容影响。

## 可配置变量

```go
// 遮罩颜色（仅用 alpha 通道确定 darken 百分比）
var ScrimColor = "rgba(0,0,0,0.5)"

// 背景底色，被 lipgloss.Darken 暗化
var DefaultBg = "#201d1d"
```

修改方式：

```go
overlay.ScrimColor = "rgba(0,0,0,0.3)"  // 30% darken
overlay.DefaultBg = "#1a1a1a"            // 适配其他主题
```

## 颜色处理（用 lipgloss，不自实现）

| 步骤 | 使用 |
|------|------|
| 解析 ScrimColor | `lipgloss.Color` + `color.RGBAModel` |
| 暗化 DefaultBg | `lipgloss.Darken` |
| 背景 ANSI 码 | `\x1b[48;2;R;G;Bm`（24-bit true color）|
| 前景暗化 | `\x1b[2m`（SGR dim）|
| reset 匹配 | 正则 `\x1b\[(0(;\d+)*)?m` |

## ANSI reset 处理

正则匹配以下类型，每个匹配后注入 dim+bg：

| 输入 | 匹配 | 替换 |
|------|------|------|
| `\x1b[0m` | ✓ | `\x1b[0m\x1b[2m<bg>` |
| `\x1b[m` | ✓ | `\x1b[m\x1b[2m<bg>` |
| `\x1b[0;31m` | ✓ | `\x1b[0;31m\x1b[2m<bg>` |
| `\x1b[31m` | ✗ | 不变（非 reset）|
| `\x1b[1;31m` | ✗ | 不变（非 reset）|
