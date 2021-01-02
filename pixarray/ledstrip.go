package pixarray

type LEDStrip interface {
	GetPixel(i int) Pixel
	SetPixel(i int, p Pixel)
	Write() error
}
