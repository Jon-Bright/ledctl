package pixarray

import (
	"fmt"
	"syscall"
	"unsafe"
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

type PixArray struct {
	dev       dev
	numPixels int
	sendBytes []byte
	pixels    []byte
	g         int
	r         int
	b         int
}

func NewPixArray(dev dev, numPixels int, spiSpeed uint32, order int) (*PixArray, error) {
	numReset := (numPixels + 31) / 32
	val := make([]byte, numPixels*3+numReset)
	offsets := offsets[order]
	pa := PixArray{dev, numPixels, val, val[:numPixels*3], offsets[0], offsets[1], offsets[2]}

	if spiSpeed != 0 {
		err := pa.setSPISpeed(spiSpeed)
		if err != nil {
			return nil, fmt.Errorf("couldn't set SPI speed: %v", err)
		}
	}

	firstReset := make([]byte, numReset)
	_, err := dev.Write(firstReset)
	if err != nil {
		return nil, fmt.Errorf("couldn't reset: %v", err)
	}
	return &pa, nil
}

const (
	_SPI_IOC_WR_MAX_SPEED_HZ = 0x40046B04
)

func (pa *PixArray) setSPISpeed(s uint32) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(pa.dev.Fd()),
		uintptr(_SPI_IOC_WR_MAX_SPEED_HZ),
		uintptr(unsafe.Pointer(&s)),
	)
	if errno == 0 {
		return nil
	}
	return errno
}

func (pa *PixArray) Write() error {
	_, err := pa.dev.Write(pa.sendBytes)
	return err
}

func (pa *PixArray) NumPixels() int {
	return pa.numPixels
}

func (pa *PixArray) GetPixels() []Pixel {
	p := make([]Pixel, pa.numPixels)
	for i, v := range pa.pixels {
		switch i % 3 {
		case pa.g:
			p[i/3].G = int(v) & 0x7f
		case pa.r:
			p[i/3].R = int(v) & 0x7f
		case pa.b:
			p[i/3].B = int(v) & 0x7f
		}
	}
	return p
}

func (pa *PixArray) GetPixel(i int) Pixel {
	return Pixel{int(pa.pixels[i*3+pa.r]) & 0x7f, int(pa.pixels[i*3+pa.g]) & 0x7f, int(pa.pixels[i*3+pa.b]) & 0x7f}
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
			pa.pixels[i*3+pa.g] = byte(0x80 | p2.G)
			pa.pixels[i*3+pa.r] = byte(0x80 | p2.R)
			pa.pixels[i*3+pa.b] = byte(0x80 | p2.B)
		} else {
			pa.pixels[i*3+pa.g] = byte(0x80 | p1.G)
			pa.pixels[i*3+pa.r] = byte(0x80 | p1.R)
			pa.pixels[i*3+pa.b] = byte(0x80 | p1.B)
		}
	}
}

func (pa *PixArray) SetPerChanAlternate(num Pixel, div int, p1 Pixel, p2 Pixel) {
	totSet := Pixel{}
	shouldSet := Pixel{}
	for i := 0; i < pa.numPixels; i++ {
		shouldSet.R += num.R
		e1 := abs((totSet.R + div) - shouldSet.R)
		e2 := abs(totSet.R - shouldSet.R)
		if e1 < e2 {
			totSet.R += div
			pa.pixels[i*3+pa.r] = byte(0x80 | p2.R)
		} else {
			pa.pixels[i*3+pa.r] = byte(0x80 | p1.R)
		}
		shouldSet.G += num.G
		e1 = abs((totSet.G + div) - shouldSet.G)
		e2 = abs(totSet.G - shouldSet.G)
		if e1 < e2 {
			totSet.G += div
			pa.pixels[i*3+pa.g] = byte(0x80 | p2.G)
		} else {
			pa.pixels[i*3+pa.g] = byte(0x80 | p1.G)
		}
		shouldSet.B += num.B
		e1 = abs((totSet.B + div) - shouldSet.B)
		e2 = abs(totSet.B - shouldSet.B)
		if e1 < e2 {
			totSet.B += div
			pa.pixels[i*3+pa.b] = byte(0x80 | p2.B)
		} else {
			pa.pixels[i*3+pa.b] = byte(0x80 | p1.B)
		}
	}
}

func (pa *PixArray) SetAll(p Pixel) {
	for i := 0; i < pa.numPixels; i++ {
		pa.pixels[i*3+pa.g] = byte(0x80 | p.G)
		pa.pixels[i*3+pa.r] = byte(0x80 | p.R)
		pa.pixels[i*3+pa.b] = byte(0x80 | p.B)
	}
}

func (pa *PixArray) SetOne(i int, p Pixel) {
	pa.pixels[i*3+pa.g] = byte(0x80 | p.G)
	pa.pixels[i*3+pa.r] = byte(0x80 | p.R)
	pa.pixels[i*3+pa.b] = byte(0x80 | p.B)
}
