package layout

const (
	InitWidth         = 80
	InitHeight        = 24
	InputHeight       = 5
	StatusBarHeight   = 1
	SpacingHeight     = 1
	PaddingHorizontal = 0
	MinWidth          = 40
	MinHeight         = 12
)

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func GetChatWidth(totalWidth int) int {
	return max1(totalWidth - PaddingHorizontal*2)
}

func GetChatHeight(totalHeight int) int {
	return max1(totalHeight - InputHeight - SpacingHeight - StatusBarHeight)
}

func GetInputWidth(totalWidth int) int {
	return max1(totalWidth - PaddingHorizontal*2)
}

func GetInputHeight() int {
	return InputHeight
}

func GetStatusBarHeight() int {
	return StatusBarHeight
}

func GetSpacingHeight() int {
	return SpacingHeight
}

func GetPaddingHorizontal() int {
	return PaddingHorizontal
}
