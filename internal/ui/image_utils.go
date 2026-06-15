package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"

	_ "golang.org/x/image/webp"
)

// circleMask generates a circular mask and applies it to the given image.
func circleMask(src image.Image) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	// Ensure square
	size := width
	if height < size {
		size = height
	}
	
	// Create a new RGBA image with transparency
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	
	radius := float64(size) / 2
	center := float64(size) / 2
	
	// Draw the source image with a circular mask
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)
	
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Calculate distance to center
			dx := float64(x) + 0.5 - center
			dy := float64(y) + 0.5 - center
			dist := math.Sqrt(dx*dx + dy*dy)
			
			if dist <= radius {
				// We can add anti-aliasing here if needed, but simple clipping is fine
				alpha := uint8(255)
				if dist > radius-1.0 {
					// anti-aliasing on the border
					alpha = uint8((radius - dist) * 255)
				}
				
				// Get source pixel
				srcX := bounds.Min.X + (width-size)/2 + x
				srcY := bounds.Min.Y + (height-size)/2 + y
				
				r, g, b, a := src.At(srcX, srcY).RGBA()
				
				// Apply alpha
				if alpha < 255 {
					a = uint32(alpha) * 0x101
					r = (r * a) / 0xffff
					g = (g * a) / 0xffff
					b = (b * a) / 0xffff
				}
				
				dst.Set(x, y, color.NRGBA64{
					R: uint16(r),
					G: uint16(g),
					B: uint16(b),
					A: uint16(a),
				})
			}
		}
	}
	return dst
}

// applyCircularMask takes raw image bytes and returns PNG bytes with a circular crop
func applyCircularMask(imgData []byte) []byte {
	if len(imgData) == 0 {
		return imgData
	}
	
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return imgData
	}
	
	masked := circleMask(img)
	
	var buf bytes.Buffer
	err = png.Encode(&buf, masked)
	if err != nil {
		return imgData
	}
	
	return buf.Bytes()
}
