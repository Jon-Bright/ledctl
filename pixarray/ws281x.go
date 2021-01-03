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
	mbox       *os.File
	mboxSize   uint32
	rp         *RasPiHW
	pixHandle  uintptr
	pixBusAddr uintptr
	pixBuf     mmap.MMap
	pixBufOffs uintptr
	dmaCb      *dmaCallback
	dmaBuf     mmap.MMap
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
	wa.initDmaCb()
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

func (ws *WS281x) GetPixel(i int) Pixel {
	return Pixel{}
}

func (ws *WS281x) SetPixel(i int, p Pixel) {
	ws.pixBuf[int(ws.pixBufOffs)+i*ws.numColors+ws.r] = byte(p.R)
	ws.pixBuf[int(ws.pixBufOffs)+i*ws.numColors+ws.g] = byte(p.G)
	ws.pixBuf[int(ws.pixBufOffs)+i*ws.numColors+ws.b] = byte(p.B)
	if ws.numColors == 4 {
		ws.pixBuf[int(ws.pixBufOffs)+i*ws.numColors+ws.w] = byte(p.W)
	}
}

func (wa *WS281x) Write() error {
	// TODO
	return nil
}
