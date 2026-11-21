package interpreter

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/ean"

	"github.com/ivantit66/onebase/internal/i18n/i18nerr"
)

const (
	barcodeModulePixels = 3
	barcodeQuietModules = 11
	barcodeHeight       = 96
	barcodeMaxWidth     = 4096
	barcodeMaxCodeBytes = 80
	gs1MaxDataBytes     = 48
)

func builtinBarcodePNG(args []any, _ string, _ int) (any, error) {
	if len(args) != 2 {
		return nil, i18nerr.New("ШтрихкодPNG: ожидаются тип и код")
	}
	kind, kindOK := args[0].(string)
	code, codeOK := args[1].(string)
	if !kindOK || !codeOK {
		return nil, i18nerr.New("ШтрихкодPNG: тип и код должны быть строками")
	}
	if len(code) == 0 || len(code) > barcodeMaxCodeBytes {
		return nil, i18nerr.New("ШтрихкодPNG: длина кода должна быть от 1 до 80 байт")
	}
	if len(kind) > 24 {
		return nil, i18nerr.New("ШтрихкодPNG: неизвестный тип штрихкода")
	}

	symbol, err := encodeLinearBarcode(strings.ToUpper(strings.TrimSpace(kind)), code)
	if err != nil {
		return nil, err
	}
	bounds := symbol.Bounds()
	width := (bounds.Dx() + 2*barcodeQuietModules) * barcodeModulePixels
	if bounds.Dx() <= 0 || width > barcodeMaxWidth {
		return nil, i18nerr.New("ШтрихкодPNG: штрихкод слишком широк для изображения")
	}
	img := image.NewPaletted(image.Rect(0, 0, width, barcodeHeight), color.Palette{color.White, color.Black})
	for column := 0; column < bounds.Dx(); column++ {
		red, _, _, _ := symbol.At(bounds.Min.X+column, bounds.Min.Y).RGBA()
		if red != 0 {
			continue
		}
		left := (barcodeQuietModules + column) * barcodeModulePixels
		for y := 0; y < barcodeHeight; y++ {
			for x := left; x < left+barcodeModulePixels; x++ {
				img.SetColorIndex(x, y, 1)
			}
		}
	}
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&buffer, img); err != nil {
		return nil, i18nerr.Wrapf(err, "ШтрихкодPNG: не удалось сформировать PNG")
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func encodeLinearBarcode(kind, code string) (barcode.Barcode, error) {
	switch kind {
	case "EAN8", "EAN-8", "EAN13", "EAN-13":
		digits := 13
		if kind == "EAN8" || kind == "EAN-8" {
			digits = 8
		}
		if len(code) != digits || strings.Trim(code, "0123456789") != "" {
			return nil, i18nerr.Errorf("ШтрихкодPNG: EAN должен содержать %d цифр", digits)
		}
		symbol, err := ean.Encode(code)
		if err != nil {
			return nil, i18nerr.New("ШтрихкодPNG: неверная контрольная цифра EAN")
		}
		return symbol, nil
	case "CODE39":
		if strings.Trim(code, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-. $/+%") != "" {
			return nil, i18nerr.New("ШтрихкодPNG: недопустимый символ Code39")
		}
		return code39.Encode(code, false, false)
	case "CODE128":
		for _, c := range code {
			if c > 127 {
				return nil, i18nerr.New("ШтрихкодPNG: Code128 принимает только символы ASCII")
			}
		}
		return code128.Encode(code)
	case "EAN128", "GS1-128":
		if len(code) <= gs1MaxDataBytes && validGS1BarcodePayload(code) {
			fnc1 := string(code128.FNC1)
			return code128.Encode(fnc1 + strings.ReplaceAll(code, "\x1d", fnc1))
		}
		return nil, i18nerr.New("ШтрихкодPNG: GS1-128 требует подготовленные данные AI длиной до 48 символов")
	default:
		return nil, i18nerr.New("ШтрихкодPNG: неизвестный тип штрихкода")
	}
}

func validGS1BarcodePayload(code string) bool {
	const gs1Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!\"%&'()*+,-./:;<=>?_"
	for part := range strings.SplitSeq(code, "\x1d") {
		if len(part) < 3 || strings.Trim(part[:2], "0123456789") != "" || strings.Trim(part, gs1Alphabet) != "" {
			return false
		}
	}
	return true
}
