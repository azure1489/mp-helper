package types

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestBodyImageRejectsIncompletePixelData(t *testing.T) {
	for _, format := range []string{"png", "jpeg"} {
		t.Run(format, func(t *testing.T) {
			var b bytes.Buffer
			im := image.NewRGBA(image.Rect(0, 0, 16, 16))
			if format == "png" {
				png.Encode(&b, im)
			} else {
				jpeg.Encode(&b, im, nil)
			}
			raw := b.Bytes()
			if err := ValidateBodyImage(raw); err != nil {
				t.Fatal(err)
			}
			cut := -1
			for i := 1; i < len(raw); i++ {
				if _, _, err := image.DecodeConfig(bytes.NewReader(raw[:i])); err == nil {
					if _, _, err = image.Decode(bytes.NewReader(raw[:i])); err != nil {
						cut = i
						break
					}
				}
			}
			if cut < 0 {
				t.Fatal("could not produce header-only fixture")
			}
			if err := ValidateBodyImage(raw[:cut]); err == nil {
				t.Fatal("header-only pixels accepted")
			}
		})
	}
}
