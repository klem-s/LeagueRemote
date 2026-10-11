// Package icon generates the PWA's app icon and manifest. It has no
// dependency on the rest of the app.
package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
)

var (
	cache   = map[int][]byte{}
	cacheMu sync.Mutex
)

// PNG renders the app icon (gold ring, teal hextech dot) directly to PNG
// via the standard image/png package — no external assets needed.
func PNG(size int) []byte {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if b, ok := cache[size]; ok {
		return b
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := float64(size) / 2
	rOut := float64(size) * 0.34
	rIn := float64(size) * 0.26
	dot := float64(size) * 0.11
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - c + 0.5
			dy := float64(y) - c + 0.5
			d := math.Hypot(dx, dy)
			var col color.RGBA
			switch {
			case d < dot:
				col = color.RGBA{10, 200, 185, 255}
			case d > rIn && d < rOut:
				col = color.RGBA{200, 170, 110, 255}
			default:
				col = color.RGBA{9, 20, 40, 255}
			}
			img.Set(x, y, col)
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	b := buf.Bytes()
	cache[size] = b
	return b
}

func Manifest() map[string]interface{} {
	return map[string]interface{}{
		"name": "LoL Remote", "short_name": "LoL Remote", "start_url": "/", "scope": "/",
		"display": "standalone", "background_color": "#091428", "theme_color": "#091428",
		"icons": []map[string]interface{}{
			{"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"},
			{"src": "/icon-512.png", "sizes": "512x512", "type": "image/png"},
		},
	}
}
