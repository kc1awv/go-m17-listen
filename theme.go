// theme.go
package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type customTheme struct{}

func (customTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return theme.DefaultTextMonospaceFont()
	}
	if s.Bold {
		if s.Italic {
			return theme.DefaultTextBoldItalicFont()
		}
		return theme.DefaultTextBoldFont()
	}
	if s.Italic {
		return theme.DefaultTextItalicFont()
	}
	return theme.DefaultTextFont()
}

func (customTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (customTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (customTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}
