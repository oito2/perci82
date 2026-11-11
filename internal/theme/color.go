// Copyright (C) 2026  oito2
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package theme

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// rgb is an opaque color with channels from 0 to 1.
type rgb struct{ r, g, b float64 }

var (
	black = rgb{0, 0, 0}
	white = rgb{1, 1, 1}
)

// parseHex parses "#RRGGBB" (or "#RGB"; an alpha pair is ignored).
func parseHex(s string) (rgb, error) {
	digits := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(digits) == 3 || len(digits) == 4 {
		var b strings.Builder
		for _, d := range digits {
			b.WriteRune(d)
			b.WriteRune(d)
		}
		digits = b.String()
	}
	if len(digits) != 6 && len(digits) != 8 {
		return rgb{}, fmt.Errorf("invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(digits[:6], 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("invalid hex color %q", s)
	}
	return rgb{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}, nil
}

// mustHex parses a color of the embedded fixtures.
func mustHex(s string) rgb {
	c, err := parseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// roundHalfAway rounds half away from zero.
func roundHalfAway(v float64) float64 {
	return math.Copysign(math.Floor(math.Abs(v)+0.5), v)
}

// channel8 converts a channel to 0–255, rounded and clamped.
func channel8(v float64) int {
	return int(min(255, max(0, roundHalfAway(v*255))))
}

// hex formats c as "#RRGGBB".
func (c rgb) hex() string {
	return fmt.Sprintf("#%02X%02X%02X", channel8(c.r), channel8(c.g), channel8(c.b))
}

// linearize converts one sRGB channel to linear light.
func linearize(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// luminance is the WCAG relative luminance of c.
func (c rgb) luminance() float64 {
	return 0.2126*linearize(c.r) + 0.7152*linearize(c.g) + 0.0722*linearize(c.b)
}

// contrastRatio is the WCAG contrast ratio of a and b, from 1 to 21.
func contrastRatio(a, b rgb) float64 {
	la, lb := a.luminance(), b.luminance()
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// minContrastRatio is the lowest contrast ratio of c against backgrounds.
func minContrastRatio(c rgb, backgrounds []rgb) float64 {
	lowest := 21.0
	for _, bg := range backgrounds {
		lowest = min(lowest, contrastRatio(c, bg))
	}
	return lowest
}

// foregroundFor returns black or white, whichever contrasts more with
// background (black on a tie).
func foregroundFor(background rgb) rgb {
	if contrastRatio(black, background) >= contrastRatio(white, background) {
		return black
	}
	return white
}

// ensureContrast returns c when it reaches target against every
// background; otherwise the first lightness step of c (0.5% each, darker
// first) that does, else black or white, whichever contrasts more.
func ensureContrast(c rgb, backgrounds []rgb, target float64) rgb {
	if minContrastRatio(c, backgrounds) >= target {
		return c
	}
	h, s, l := toHSL(c)
	for step := 1; step <= 200; step++ {
		for _, sign := range []float64{-1, 1} {
			lightness := l + sign*float64(step)*0.005
			if lightness < 0 || lightness > 1 {
				continue
			}
			candidate := fromHSL(h, s, lightness)
			if minContrastRatio(candidate, backgrounds) >= target {
				return candidate
			}
		}
	}
	if minContrastRatio(black, backgrounds) >= minContrastRatio(white, backgrounds) {
		return black
	}
	return white
}

// toHSL converts c, rounded to 8-bit channels, to hue (degrees),
// saturation and lightness.
func toHSL(c rgb) (h, s, l float64) {
	r := float64(channel8(c.r)) / 255
	g := float64(channel8(c.g)) / 255
	b := float64(channel8(c.b)) / 255
	hi, lo := max(r, g, b), min(r, g, b)
	delta := hi - lo
	switch {
	case hi == 0 || delta == 0:
		h = 0
	case hi == r:
		rem := math.Mod((g-b)/delta, 6)
		if rem < 0 {
			rem += 6
		}
		h = 60 * rem
	case hi == g:
		h = 60 * ((b-r)/delta + 2)
	default:
		h = 60 * ((r-g)/delta + 4)
	}
	l = (hi + lo) / 2
	if hi != lo {
		s = min(1, max(0, delta/(1-math.Abs(2*l-1))))
	}
	return h, s, l
}

// fromHSL converts hue, saturation and lightness to a color with 8-bit
// channels.
func fromHSL(h, s, l float64) rgb {
	chroma := (1 - math.Abs(2*l-1)) * s
	second := chroma * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - chroma/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = chroma, second, 0
	case h < 120:
		r, g, b = second, chroma, 0
	case h < 180:
		r, g, b = 0, chroma, second
	case h < 240:
		r, g, b = 0, second, chroma
	case h < 300:
		r, g, b = second, 0, chroma
	default:
		r, g, b = chroma, 0, second
	}
	ch := func(v float64) float64 { return roundHalfAway((v+m)*255) / 255 }
	return rgb{ch(r), ch(g), ch(b)}
}
