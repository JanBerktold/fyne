package gl

import (
	"image"
	"sync"

	"fyne.io/fyne"
	"fyne.io/fyne/theme"
	"github.com/goki/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type compositeFace struct {
	sync.Mutex

	chosen, fallback         font.Face
	chosenFont, fallbackFont *truetype.Font
}

func (c *compositeFace) containsGlyph(font *truetype.Font, r rune) bool {
	c.Lock()
	defer c.Unlock()

	return font != nil && font.Index(r) != 0
}

func (c *compositeFace) Close() error {
	if c.chosen != nil {
		_ = c.chosen.Close()
	}

	return c.fallback.Close()
}

func (c *compositeFace) Glyph(dot fixed.Point26_6, r rune) (
	dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	contains := c.containsGlyph(c.chosenFont, r)

	c.Lock()
	defer c.Unlock()

	if contains {
		return c.chosen.Glyph(dot, r)
	}

	return c.fallback.Glyph(dot, r)
}

func (c *compositeFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	contains := c.containsGlyph(c.chosenFont, r)

	c.Lock()
	defer c.Unlock()

	if contains {
		c.chosen.GlyphBounds(r)
	}
	return c.fallback.GlyphBounds(r)
}

func (c *compositeFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	contains := c.containsGlyph(c.chosenFont, r)

	c.Lock()
	defer c.Unlock()

	if contains {
		return c.chosen.GlyphAdvance(r)
	}
	return c.fallback.GlyphAdvance(r)
}

func (c *compositeFace) Kern(r0, r1 rune) fixed.Int26_6 {
	contains0 := c.containsGlyph(c.chosenFont, r0)
	contains1 := c.containsGlyph(c.chosenFont, r1)

	c.Lock()
	defer c.Unlock()

	if contains0 && contains1 {
		return c.chosen.Kern(r0, r1)
	}
	return c.fallback.Kern(r0, r1)
}

func (c *compositeFace) Metrics() font.Metrics {
	c.Lock()
	defer c.Unlock()

	return c.chosen.Metrics()
}

func newFontWithFallback(chosen, fallback font.Face, chosenFont, fallbackFont *truetype.Font) font.Face {
	return &compositeFace{chosen: chosen, fallback: fallback, chosenFont: chosenFont, fallbackFont: fallbackFont}
}

type fontCacheItem struct {
	font, fallback *truetype.Font
	faces          map[truetype.Options]font.Face
}

var fontCache = make(map[fyne.TextStyle]*fontCacheItem)

func cachedFontFace(style fyne.TextStyle, opts *truetype.Options) font.Face {
	comp := fontCache[style]

	if comp == nil {
		var f1, f2 *truetype.Font
		switch {
		case style.Monospace:
			f1 = loadFont(theme.TextMonospaceFont())
			f2 = loadFont(theme.DefaultTextMonospaceFont())
		case style.Bold:
			if style.Italic {
				f1 = loadFont(theme.TextBoldItalicFont())
				f2 = loadFont(theme.DefaultTextBoldItalicFont())
			} else {
				f1 = loadFont(theme.TextBoldFont())
				f2 = loadFont(theme.DefaultTextBoldFont())
			}
		case style.Italic:
			f1 = loadFont(theme.TextItalicFont())
			f2 = loadFont(theme.DefaultTextItalicFont())
		default:
			f1 = loadFont(theme.TextFont())
			f2 = loadFont(theme.DefaultTextFont())
		}

		comp = &fontCacheItem{font: f1, fallback: f2, faces: make(map[truetype.Options]font.Face)}
		fontCache[style] = comp
	}

	face := comp.faces[*opts]
	if face == nil {
		f1 := truetype.NewFace(comp.font, opts)
		f2 := truetype.NewFace(comp.fallback, opts)
		face = newFontWithFallback(f1, f2, comp.font, comp.fallback)

		comp.faces[*opts] = face
	}

	return face
}

func clearFontCache() {
	for _, item := range fontCache {
		for _, face := range item.faces {
			err := face.Close()

			if err != nil {
				fyne.LogError("failed to close font face", err)
				return
			}
		}
	}

	fontCache = make(map[fyne.TextStyle]*fontCacheItem)
}
