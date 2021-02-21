package pixarray

import (
	"fmt"
	rpi "github.com/Jon-Bright/ledctl/rpi"
)

type LPD8806 struct {
	numColors int
	pixels    []byte
	g         int
	r         int
	b         int
	w         int
	dev       dev
	sendBytes []byte
	rp        *rpi.RPi
}

func NewLPD8806(dev dev, numPixels int, numColors int, spiSpeed uint32, order int) (LEDStrip, error) {
	numReset := (numPixels + 31) / 32
	val := make([]byte, numPixels*numColors+numReset)
	offsets := offsets[order]
	rp, err := rpi.NewRPi()
	if err != nil {
		return nil, fmt.Errorf("couldn't make RPi: %v", err)
	}
	la := LPD8806{numColors, val[:numPixels*numColors], offsets[0], offsets[1], offsets[2], offsets[3], dev, val, rp}

	if spiSpeed != 0 {
		err := rp.SetSPISpeed(la.dev.Fd(), spiSpeed)
		if err != nil {
			return nil, fmt.Errorf("couldn't set SPI speed: %v", err)
		}
	}

	firstReset := make([]byte, numReset)
	_, err = dev.Write(firstReset)
	if err != nil {
		return nil, fmt.Errorf("couldn't reset: %v", err)
	}
	return &la, nil
}

func (la *LPD8806) RPi() *rpi.RPi {
	return la.rp
}

func (la *LPD8806) MaxPerChannel() int {
	return 127
}

func (la *LPD8806) Write() error {
	_, err := la.dev.Write(la.sendBytes)
	return err
}

func (la *LPD8806) GetPixel(i int) Pixel {
	if la.numColors == 4 {
		return Pixel{
			int(la.pixels[i*la.numColors+la.r]) & 0x7f,
			int(la.pixels[i*la.numColors+la.g]) & 0x7f,
			int(la.pixels[i*la.numColors+la.b]) & 0x7f,
			int(la.pixels[i*la.numColors+la.w]) & 0x7f,
		}
	}
	return Pixel{
		int(la.pixels[i*la.numColors+la.r]) & 0x7f,
		int(la.pixels[i*la.numColors+la.g]) & 0x7f,
		int(la.pixels[i*la.numColors+la.b]) & 0x7f,
		-1,
	}
}

func (la *LPD8806) SetPixel(i int, p Pixel) {
	la.pixels[i*la.numColors+la.r] = byte(0x80 | p.R)
	la.pixels[i*la.numColors+la.g] = byte(0x80 | p.G)
	la.pixels[i*la.numColors+la.b] = byte(0x80 | p.B)
	if la.numColors == 4 {
		la.pixels[i*la.numColors+la.w] = byte(0x80 | p.W)
	}
}
