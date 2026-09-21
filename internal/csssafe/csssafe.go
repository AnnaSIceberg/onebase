package csssafe

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	hexColorRe    = regexp.MustCompile(`(?i)^#(?:[0-9a-f]{3}|[0-9a-f]{4}|[0-9a-f]{6}|[0-9a-f]{8})$`)
	rgbFunctionRe = regexp.MustCompile(`(?i)^(rgb|rgba)\((.*)\)$`)
	cssNumberRe   = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)$`)
	lengthRe      = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?(?:px|pt|mm|cm|in|em|rem|%)$`)
)

var namedColors = map[string]bool{
	"aliceblue": true, "antiquewhite": true, "aquamarine": true, "azure": true,
	"beige": true, "bisque": true, "blanchedalmond": true,
	"black": true, "white": true, "red": true, "green": true, "blue": true,
	"yellow": true, "orange": true, "purple": true, "gray": true, "grey": true,
	"silver": true, "maroon": true, "olive": true, "lime": true, "aqua": true,
	"teal": true, "navy": true, "fuchsia": true, "magenta": true, "cyan": true,
	"pink": true, "brown": true, "gold": true, "transparent": true,
	"darkred": true, "darkgreen": true, "darkblue": true, "lightgray": true,
	"lightgrey": true, "lightblue": true, "lightgreen": true, "lightyellow": true,
	"chocolate": true, "coral": true, "cornflowerblue": true, "cornsilk": true,
	"crimson": true, "darkcyan": true, "darkgoldenrod": true, "darkgray": true,
	"darkgrey": true, "darkkhaki": true, "darkmagenta": true, "darkolivegreen": true,
	"darkorange": true, "darkorchid": true, "darksalmon": true, "darkseagreen": true,
	"darkslateblue": true, "darkslategray": true, "darkslategrey": true,
	"darkturquoise": true, "darkviolet": true, "deeppink": true, "deepskyblue": true,
	"dimgray": true, "dimgrey": true, "dodgerblue": true, "firebrick": true,
	"floralwhite": true, "forestgreen": true, "gainsboro": true, "ghostwhite": true,
	"goldenrod": true, "greenyellow": true, "honeydew": true, "hotpink": true,
	"indianred": true, "indigo": true, "ivory": true, "khaki": true, "lavender": true,
	"lavenderblush": true, "lawngreen": true, "lemonchiffon": true, "lightcoral": true,
	"lightcyan": true, "lightgoldenrodyellow": true, "lightpink": true,
	"lightsalmon": true, "lightseagreen": true, "lightskyblue": true,
	"lightslategray": true, "lightslategrey": true, "lightsteelblue": true,
	"limegreen": true, "linen": true, "mediumaquamarine": true, "mediumblue": true,
	"mediumorchid": true, "mediumpurple": true, "mediumseagreen": true,
	"mediumslateblue": true, "mediumspringgreen": true, "mediumturquoise": true,
	"mediumvioletred": true, "midnightblue": true, "mintcream": true,
	"mistyrose": true, "moccasin": true, "navajowhite": true, "oldlace": true,
	"olivedrab": true, "orangered": true, "orchid": true, "palegoldenrod": true,
	"palegreen": true, "paleturquoise": true, "palevioletred": true,
	"papayawhip": true, "peachpuff": true, "peru": true, "plum": true,
	"powderblue": true, "rebeccapurple": true, "rosybrown": true, "royalblue": true,
	"saddlebrown": true, "salmon": true, "sandybrown": true, "seagreen": true,
	"seashell": true, "sienna": true, "skyblue": true, "slateblue": true,
	"slategray": true, "slategrey": true, "snow": true, "springgreen": true,
	"steelblue": true, "tan": true, "thistle": true, "tomato": true,
	"turquoise": true, "violet": true, "wheat": true, "whitesmoke": true,
	"yellowgreen": true,
}

// Color returns v only when it is a constrained CSS color value suitable for
// inline styles.
func Color(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	switch {
	case hexColorRe.MatchString(v):
		return v
	case validLegacyRGBColor(v):
		return v
	case namedColors[strings.ToLower(v)]:
		return v
	}
	return ""
}

// validLegacyRGBColor проверяет comma-separated синтаксис rgb()/rgba().
// Одной проверки набора символов недостаточно: она принимала rgb(,), rgba(1)
// и значения вне диапазона, которые браузер затем молча отбрасывал. Каналы
// допускают числа 0..255 или проценты 0..100, alpha — 0..1 или 0..100%.
func validLegacyRGBColor(v string) bool {
	m := rgbFunctionRe.FindStringSubmatch(v)
	if m == nil {
		return false
	}
	parts := strings.Split(m[2], ",")
	want := 3
	if strings.EqualFold(m[1], "rgba") {
		want = 4
	}
	if len(parts) != want {
		return false
	}
	for i := 0; i < 3; i++ {
		if !cssColorNumberInRange(parts[i], 255) {
			return false
		}
	}
	return want == 3 || cssColorNumberInRange(parts[3], 1)
}

func cssColorNumberInRange(raw string, max float64) bool {
	s := strings.TrimSpace(raw)
	percent := strings.HasSuffix(s, "%")
	if percent {
		s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
		max = 100
	}
	if !cssNumberRe.MatchString(s) {
		return false
	}
	n, err := strconv.ParseFloat(s, 64)
	return err == nil && n >= 0 && n <= max
}

// Length returns v only when it is a simple CSS length used by layout previews.
func Length(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.EqualFold(v, "auto") {
		return "auto"
	}
	if v == "0" || lengthRe.MatchString(v) {
		return v
	}
	return ""
}

// FontFamily strips CSS-breaking characters from a font-family value.
func FontFamily(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\'', '<', '>', ';', '\\':
			return -1
		default:
			return r
		}
	}, v)
	return strings.TrimSpace(v)
}

// TextAlign returns one of the allowed text-align values.
func TextAlign(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "left", "right", "center", "justify":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return ""
	}
}
