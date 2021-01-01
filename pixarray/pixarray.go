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
	GRB: {0, 1, 2},
	BRG: {2, 1, 0},
	BGR: {1, 2, 0},
	GBR: {0, 2, 1},
	RGB: {1, 0, 2},
	RBG: {2, 0, 1},
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
}

func (p *Pixel) String() string {
	return fmt.Sprintf("%02x%02x%02x", p.R, p.G, p.B)
}

// This is satisfied by os.File, but this minimal interface makes testing easier
type dev interface {
	Fd() uintptr
	Write(b []byte) (n int, err error)
}

type PixArray interface {
	NumPixels() int
	GetPixel(i int) Pixel
	GetPixels() []Pixel
	SetAll(p Pixel)
	SetOne(i int, p Pixel)
	SetPerChanAlternate(num Pixel, div int, p1 Pixel, p2 Pixel)
	Write() error
}
