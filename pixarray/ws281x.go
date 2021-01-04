package pixarray

import (
	"fmt"
	mmap "github.com/edsrzf/mmap-go"
	"os"
)

type WS281x struct {
	numPixels  int
	numColors  int
	g          int
	r          int
	b          int
	w          int
	pixels     []byte
	mbox       *os.File
	mboxSize   uint32
	rp         *RasPiHW
	pixHandle  uintptr
	pixBusAddr uintptr
	pixBuf     mmap.MMap
	pixBufUint []uint32
	pixBufOffs uintptr
	dmaCb      *dmaControl
	dmaBuf     mmap.MMap
	dmaBufOffs uintptr
	dma        *dmaT
	pwmBuf     mmap.MMap
	pwm        *pwmT
	gpioBuf    mmap.MMap
	gpio       *gpioT
	cmClkBuf   mmap.MMap
	cmClk      *cmClkT
}

func NewWS281x(numPixels int, numColors int, order int, freq uint, dma int, pins []int) (LEDStrip, error) {
	rp, err := detectRPiHW()
	if err != nil {
		return nil, fmt.Errorf("couldn't detect RPi hardware: %v", err)
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
	wa.mbox, err = wa.mboxOpen()
	if err != nil {
		return nil, fmt.Errorf("couldn't open mbox: %v", err)
	}
	wa.calcMboxSize(freq)
	wa.pixHandle, err = wa.allocMem()
	if err != nil {
		return nil, fmt.Errorf("couldn't allocMem: %v", err)
	}
	fmt.Printf("got handle %08X\n", wa.pixHandle)
	wa.pixBusAddr, err = wa.lockMem(wa.pixHandle)
	if err != nil {
		wa.freeMem(wa.pixHandle) // Ignore error
		return nil, fmt.Errorf("couldn't lockMem: %v", err)
	}
	fmt.Printf("got busAddr %08X\n", wa.pixBusAddr)
	wa.pixBuf, wa.pixBufOffs, err = wa.mapMem(wa.busToPhys(wa.pixBusAddr), int(wa.mboxSize))
	if err != nil {
		wa.unlockMem(wa.pixHandle) // Ignore error
		wa.freeMem(wa.pixHandle)   // Ignore error
		return nil, fmt.Errorf("couldn't map pixBuf: %v", err)
	}
	fmt.Printf("got offset %d\n", wa.pixBufOffs)
	wa.pixBufUint = mmapToUintSlice(wa.pixBuf)
	wa.initDmaControl()
	err = wa.mapDmaRegisters(dma)
	if err != nil {
		wa.unlockMem(wa.pixHandle) // Ignore error
		wa.freeMem(wa.pixHandle)   // Ignore error
		return nil, fmt.Errorf("couldn't init registers: %v", err)
	}
	err = wa.initGpio(pins)
	if err != nil {
		wa.unlockMem(wa.pixHandle) // Ignore error
		wa.freeMem(wa.pixHandle)   // Ignore error
		return nil, fmt.Errorf("couldn't init GPIO: %v", err)
	}
	wa.initPwm(freq)

	return &wa, nil
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
	pbOffs := (uintptr(ws.dmaCb.sourceAd) - ws.pixBusAddr) / 4

	// We need to wait for DMA to be done before we start touching the buffer it's outputting
	err := ws.waitForDMAEnd()
	if err != nil {
		return fmt.Errorf("pre-DMA wait failed: %v", err)
	}

	// TODO: channels, do properly - this just assumes both channels show the same thing
	for c := 0; c < 2; c++ {
		rpPos := pbOffs + uintptr(c)
		bitPos := 31
		for i := 0; i < ws.numPixels; i++ {
			for j := 0; j < ws.numColors; j++ {
				for k := 7; k >= 0; k-- {
					symbol := SYMBOL_LOW
					if (ws.pixels[i*ws.numColors+j] & (1 << uint(k))) != 0 {
						symbol = SYMBOL_HIGH
					}
					for l := 2; l >= 0; l-- {
						ws.pixBufUint[rpPos] &= ^(1 << uint(bitPos))
						if (symbol & (1 << uint(l))) != 0 {
							ws.pixBufUint[rpPos] |= 1 << uint(bitPos)
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
	ws.startDma()
	return nil
}
