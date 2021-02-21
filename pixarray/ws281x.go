package pixarray

import (
	"fmt"
	rpi "github.com/Jon-Bright/ledctl/rpi"
)

type WS281x struct {
	numPixels  int
	numColors  int
	g          int
	r          int
	b          int
	w          int
	pixels     []byte
	pixDMA     *rpi.DMABuf
	pixDMAUint []uint32
	rp         *rpi.RPi
}

const (
	LED_RESET_US = 55
)

func NewWS281x(numPixels int, numColors int, order int, freq uint, dma int, pins []int) (LEDStrip, error) {
	rp, err := rpi.NewRPi()
	if err != nil {
		return nil, fmt.Errorf("couldn't init RPi: %v", err)
	}
	offsets := offsets[order]
	wa := WS281x{
		numPixels: numPixels,
		numColors: numColors,
		g:         offsets[0],
		r:         offsets[1],
		b:         offsets[2],
		w:         offsets[3],
		pixels:    make([]byte, numPixels*numColors),
		rp:        rp,
	}
	bytes := wa.pwmByteCount(freq)
	wa.pixDMA, err = rp.GetDMABuf(bytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't get DMA buffer: %v", err)
	}
	wa.pixDMAUint = wa.pixDMA.Uint32Slice()
	err = rp.InitDMA(dma)
	if err != nil {
		rp.FreeDMABuf(wa.pixDMA) // Ignore error
		return nil, fmt.Errorf("couldn't init registers: %v", err)
	}
	err = rp.InitGPIO()
	if err != nil {
		rp.FreeDMABuf(wa.pixDMA) // Ignore error
		return nil, fmt.Errorf("couldn't init GPIO: %v", err)
	}
	err = rp.InitPWM(freq, wa.pixDMA, bytes, pins)
	if err != nil {
		rp.FreeDMABuf(wa.pixDMA) // Ignore error
		return nil, fmt.Errorf("couldn't init PWM: %v", err)
	}

	return &wa, nil
}

// pwmByteCount calculates the number of bytes needed to store the data for PWM to send - three
// bits per WS281x bit, plus enough bits to provide an appropriate reset time afterwards at the
// given frequency. It returns that byte count.
func (ws *WS281x) pwmByteCount(freq uint) uint {
	// Every bit transmitted needs 3 bits of buffer, because bits are transmitted as
	// ‾|__ (0) or ‾‾|_ (1). Each color of each pixel needs 8 "real" bits.
	bits := uint(3 * ws.numColors * ws.numPixels * 8)

	// freq is typically 800kHz, so for LED_RESET_US=55 us, this gives us
	// ((55 * (800000 * 3)) / 1000000
	// ((55 * 2400000) / 1000000
	// 132000000 / 1000000
	// 132
	// Taking this the other way, 132 bits of buffer is 132/3=44 "real" bits. With each "real" bit
	// taking 1/800000th of a second, this will take 44/800000ths of a second, which is
	// 0.000055s - 55 us.
	bits += ((LED_RESET_US * (freq * 3)) / 1000000)

	// This isn't a PDP-11, so there are 8 bits in a byte
	bytes := bits / 8

	// Round up to next uint32
	bytes -= bytes % 4
	bytes += 4

	bytes *= rpi.RPI_PWM_CHANNELS

	return bytes
}

func (ws *WS281x) RPi() *rpi.RPi {
	return ws.rp
}

func (ws *WS281x) MaxPerChannel() int {
	return 255
}

func (ws *WS281x) GetPixel(i int) Pixel {
	p := Pixel{int(ws.pixels[i*ws.numColors+ws.r]), int(ws.pixels[i*ws.numColors+ws.g]), int(ws.pixels[i*ws.numColors+ws.b]), -1}
	if ws.numColors == 4 {
		p.W = int(ws.pixels[i*ws.numColors+ws.w])
	}
	return p
}

func (ws *WS281x) SetPixel(i int, p Pixel) {
	ws.pixels[i*ws.numColors+ws.r] = byte(p.R)
	ws.pixels[i*ws.numColors+ws.g] = byte(p.G)
	ws.pixels[i*ws.numColors+ws.b] = byte(p.B)
	if ws.numColors == 4 {
		ws.pixels[i*ws.numColors+ws.w] = byte(p.W)
	}
}

const (
	SYMBOL_HIGH = 0x6 // 1 1 0
	SYMBOL_LOW  = 0x4 // 1 0 0
)

func (ws *WS281x) Write() error {

	// We need to wait for DMA to be done before we start touching the buffer it's outputting
	err := ws.rp.WaitForDMAEnd()
	if err != nil {
		return fmt.Errorf("pre-DMA wait failed: %v", err)
	}

	// TODO: channels, do properly - this just assumes both channels show the same thing
	for c := 0; c < 2; c++ {
		rpPos := c
		bitPos := 31
		for i := 0; i < ws.numPixels; i++ {
			for j := 0; j < ws.numColors; j++ {
				for k := 7; k >= 0; k-- {
					symbol := SYMBOL_LOW
					if (ws.pixels[i*ws.numColors+j] & (1 << uint(k))) != 0 {
						symbol = SYMBOL_HIGH
					}
					for l := 2; l >= 0; l-- {
						ws.pixDMAUint[rpPos] &= ^(1 << uint(bitPos))
						if (symbol & (1 << uint(l))) != 0 {
							ws.pixDMAUint[rpPos] |= 1 << uint(bitPos)
						}
						bitPos--
						if bitPos < 0 {
							rpPos += 2
							bitPos = 31
						}
					}
				}
			}
		}
	}
	ws.rp.StartDMA(ws.pixDMA)
	return nil
}
