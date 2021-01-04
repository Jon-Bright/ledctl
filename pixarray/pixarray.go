package pixarray

import (
	"fmt"
)

const (
	GRB = iota
	BRG
	BGR
	GBR
	RGB
	RBG
)

var StringOrders map[string]int = map[string]int{
	"GRB": GRB,
	"BRG": BRG,
	"BGR": BGR,
	"GBR": GBR,
	"RGB": RGB,
	"RBG": RBG,
}

var offsets map[int][]int = map[int][]int{
	GRB: {0, 1, 2, -1},
	BRG: {2, 1, 0, -1},
	BGR: {1, 2, 0, -1},
	GBR: {0, 2, 1, -1},
	RGB: {1, 0, 2, -1},
	RBG: {2, 0, 1, -1},
}

func abs(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

type Pixel struct {
	R int
	G int
	B int
	W int
}

func (p *Pixel) String() string {
	if p.W != -1 {
		return fmt.Sprintf("%02x%02x%02x%02x", p.R, p.G, p.B, p.W)
	}
	return fmt.Sprintf("%02x%02x%02x", p.R, p.G, p.B)
}

// This is satisfied by os.File, but this minimal interface makes testing easier
type dev interface {
	Fd() uintptr
	Write(b []byte) (n int, err error)
}

type PixArray struct {
	numPixels int
	numColors int
	leds      LEDStrip
}

func NewPixArray(numPixels int, numColors int, leds LEDStrip) *PixArray {
	return &PixArray{numPixels, numColors, leds}
}

func (pa *PixArray) NumPixels() int {
	return pa.numPixels
}

func (pa *PixArray) NumColors() int {
	return pa.numColors
}

func (pa *PixArray) MaxPerChannel() int {
	return pa.leds.MaxPerChannel()
}

func (pa *PixArray) Write() error {
	return pa.leds.Write()
}

func (pa *PixArray) GetPixels() []Pixel {
	p := make([]Pixel, pa.numPixels)
	for i := 0; i < pa.numPixels; i++ {
		p[i] = pa.leds.GetPixel(i)
	}
	return p
}

func (pa *PixArray) GetPixel(i int) Pixel {
	return pa.leds.GetPixel(i)
}

func (pa *PixArray) SetAlternate(num int, div int, p1 Pixel, p2 Pixel) {
	totSet := 0
	shouldSet := 0
	for i := 0; i < pa.numPixels; i++ {
		shouldSet += num
		e1 := abs((totSet + div) - shouldSet)
		e2 := abs(totSet - shouldSet)
		if e1 < e2 {
			totSet += div
			pa.leds.SetPixel(i, p2)
		} else {
			pa.leds.SetPixel(i, p1)
		}
	}
}

func (pa *PixArray) SetPerChanAlternate(num Pixel, div int, p1 Pixel, p2 Pixel) {
	totSet := Pixel{}
	shouldSet := Pixel{}
	p := Pixel{}
	for i := 0; i < pa.numPixels; i++ {
		shouldSet.R += num.R
		e1 := abs((totSet.R + div) - shouldSet.R)
		e2 := abs(totSet.R - shouldSet.R)
		if e1 < e2 {
			totSet.R += div
			p.R = p2.R
		} else {
			p.R = p1.R
		}
		shouldSet.G += num.G
		e1 = abs((totSet.G + div) - shouldSet.G)
		e2 = abs(totSet.G - shouldSet.G)
		if e1 < e2 {
			totSet.G += div
			p.G = p2.G
		} else {
			p.G = p1.G
		}
		shouldSet.B += num.B
		e1 = abs((totSet.B + div) - shouldSet.B)
		e2 = abs(totSet.B - shouldSet.B)
		if e1 < e2 {
			totSet.B += div
			p.B = p2.B
		} else {
			p.B = p1.B
		}
		if pa.numColors == 4 {
			shouldSet.W += num.W
			e1 = abs((totSet.W + div) - shouldSet.W)
			e2 = abs(totSet.W - shouldSet.W)
			if e1 < e2 {
				totSet.W += div
				p.W = p2.W
			} else {
				p.W = p1.W
			}
		}
		pa.leds.SetPixel(i, p)
	}
}

func (pa *PixArray) SetAll(p Pixel) {
	for i := 0; i < pa.numPixels; i++ {
		pa.leds.SetPixel(i, p)
	}
}

func (pa *PixArray) SetOne(i int, p Pixel) {
	pa.leds.SetPixel(i, p)
}
