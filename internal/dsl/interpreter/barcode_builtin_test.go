package interpreter_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/sheet"
	"github.com/stretchr/testify/require"
)

func TestBarcodePNGInDSL(t *testing.T) {
	codeGS1 := "01095060001343521726063010LOT-7" + string(rune(29)) + "21ABC"
	cases := []struct{ kind, code string }{
		{"EAN8", "96385074"},
		{"EAN13", "4006381333931"},
		{"Code39", "ITEM-007"},
		{"Code128", codeGS1},
		{"EAN128", codeGS1},
		{"GS1-128", "010950600013435210" + strings.Repeat("A", 20) + "\x1d21" + strings.Repeat("B", 8)},
	}
	images := make(map[string]image.Image)
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			value := evalExpr1136(t, "ШтрихкодPNG(Вид, Код)", map[string]any{"Вид": tc.kind, "Код": tc.code})
			uri, ok := value.(string)
			require.True(t, ok)
			encoded, found := strings.CutPrefix(uri, "data:image/png;base64,")
			require.True(t, found)
			raw, err := base64.StdEncoding.DecodeString(encoded)
			require.NoError(t, err)
			img, err := png.Decode(bytes.NewReader(raw))
			require.NoError(t, err)
			images[tc.kind] = img
			bounds := img.Bounds()
			require.Equal(t, 96, bounds.Dy())
			require.LessOrEqual(t, bounds.Dx(), 4096)
			for x := 0; x < 33; x++ {
				require.Equal(t, color.Gray{Y: 255}, color.GrayModel.Convert(img.At(x, 48)))
				require.Equal(t, color.Gray{Y: 255}, color.GrayModel.Convert(img.At(bounds.Dx()-1-x, 48)))
			}
			require.Equal(t, color.Gray{Y: 0}, color.GrayModel.Convert(img.At(33, 48)))

			doc := sheet.NewDocument()
			cell := doc.GetOrCreateCell(0, 0)
			cell.Picture, cell.Width, cell.Height = uri, 140, 60
			require.Contains(t, doc.HTMLString(), "<img src=\""+uri+"\"")
			pdf, err := doc.PDF(sheet.PDFOptions{})
			require.NoError(t, err)
			require.Contains(t, string(pdf), "/Subtype /Image")
		})
	}
	// Code 128 uses 11 modules per codeword. Codeword 102 is FNC1 and
	// has run lengths 4-1-1-1-3-1. Check it at the symbol level, not metadata.
	const fnc1 = "11110101110"
	codewords := func(img image.Image) []string {
		var words []string
		x := 33
		for x+33 <= img.Bounds().Dx()-33 {
			var bits strings.Builder
			for module := 0; module < 11; module++ {
				if color.GrayModel.Convert(img.At(x+3*module, 48)) == (color.Gray{Y: 0}) {
					bits.WriteByte('1')
				} else {
					bits.WriteByte('0')
				}
			}
			words = append(words, bits.String())
			x += 33
		}
		// Omit the check codeword and the first 11 modules of the stop code.
		return words[1 : len(words)-2]
	}
	gs1Words := codewords(images["EAN128"])
	require.Equal(t, fnc1, gs1Words[0])
	require.Equal(t, 2, strings.Count(strings.Join(gs1Words, "|"), fnc1))
	require.NotContains(t, codewords(images["Code128"]), fnc1)
}

func TestBarcodePNGDSLRejectsInvalidInput(t *testing.T) {
	const src = `Функция Т()
	Попытка
		Возврат ШтрихкодPNG(Вид, Код);
	Исключение
		Возврат ОписаниеОшибки();
	КонецПопытки;
КонецФункции`
	for _, tc := range []struct{ kind, code, want string }{
		{"EAN8", "4006381333931", "EAN должен содержать 8 цифр"},
		{"EAN8", "9638507X", "EAN должен содержать 8 цифр"},
		{"EAN13", "4006381333932", "неверная контрольная цифра"},
		{"Code39", "A*B", "недопустимый символ Code39"},
		{"Code128", "деталь", "только символы ASCII"},
		{"EAN128", "(01)09506000134352", "подготовленные данные AI"},
		{"EAN128", "10LOT\x1d", "подготовленные данные AI"},
		{"GS1-128", strings.Repeat("1", 49), "подготовленные данные AI"},
		{"Code128", strings.Repeat("1", 81), "длина кода"},
		{"QR", "123", "неизвестный тип"},
	} {
		t.Run(tc.kind+"/"+tc.want, func(t *testing.T) {
			value := evalWithVars(t, src, map[string]any{"Вид": tc.kind, "Код": tc.code})
			require.Contains(t, value, tc.want)
		})
	}
}
