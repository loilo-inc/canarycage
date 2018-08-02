package main

func RunningIcon() func() string {
	icons := []string{"️🏃🏼‍♀", "️🏃🏽‍♀", "️🏃🏾‍♀", "️🏃🏿‍♀️"}
	i := -1
	return func() string {
		if i++; i >= len(icons) {
			i = 0
		}
		return icons[i]
	}
}

func MoonIcon() func(float64) string {
	return func(progress float64) string {
		if progress == 0 {
			return "🌑"
		} else if 0 < progress && progress < 0.5 {
			return "🌒"
		} else if 0.5 <= progress && progress < 0.75 {
			return "🌓"
		} else if 0.75 <= progress && progress < 1 {
			return "🌔"
		} else {
			return "🌕"
		}
	}
}