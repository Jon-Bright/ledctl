package pixarray

import (
	"fmt"
	"os"
)

type WS281x struct {
	numPixels int
	numColors int
	g         int
	r         int
	b         int
	w         int
	mbox      *os.File
	mboxSize  uint32
	rp        *RasPiHW
}

func NewWS281x(numPixels int, numColors int, order int, freq uint) (LEDStrip, error) {
	rp, err := detectRPiHW()
	if err != nil {
		return nil, fmt.Errorf("couldn't detect RPi hardware: %v", err)
	}
	offsets := offsets[order]
	wa := WS281x{numPixels, numColors, offsets[0], offsets[1], offsets[2], offsets[3], nil, 0, rp}
	wa.mbox, err = wa.mboxOpen()
	if err != nil {
		return nil, fmt.Errorf("couldn't open mbox: %v", err)
	}
	wa.calcMboxSize(freq)
	addr, err := wa.allocMem()
	if err != nil {
		return nil, fmt.Errorf("couldn't allocMem: %v", err)
	}
	fmt.Printf("got addr %08X\n", addr)
	return &wa, nil
}

func (ws *WS281x) GetPixel(i int) Pixel {
	return Pixel{}
}

func (ws *WS281x) SetPixel(i int, p Pixel) {
}

func (wa *WS281x) Write() error {
	// TODO
	return nil
}
