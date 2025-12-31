package tui

func boolToIcon(b bool) string {
	i := "✗"
	if b {
		i = "✓"
	}

	return i
}
