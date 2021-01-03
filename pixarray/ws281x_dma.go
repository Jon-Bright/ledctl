package pixarray

import (
	"fmt"
	"unsafe"
)

const (
	PWM_OFFSET    = uintptr(0x0020c000)
	GPIO_OFFSET   = uintptr(0x00200000)
	CM_PWM_OFFSET = uintptr(0x001010a0)
)

var dmaOffsets = map[int]uintptr{
	0:  0x00007000,
	1:  0x00007100,
	2:  0x00007200,
	3:  0x00007300,
	4:  0x00007400,
	5:  0x00007500,
	6:  0x00007600,
	7:  0x00007700,
	8:  0x00007800,
	9:  0x00007900,
	10: 0x00007a00,
	11: 0x00007b00,
	12: 0x00007c00,
	13: 0x00007d00,
	14: 0x00007e00,
	15: 0x00e05000,
}

type dmaT struct {
	cs        uint32
	conblkAd  uint32
	ti        uint32
	sourceAd  uint32
	destAd    uint32
	txLen     uint32
	stride    uint32
	nextConBk uint32
	debug     uint32
}

type pwmT struct {
	ctl        uint32
	sta        uint32
	dmac       uint32
	resvd_0x0c uint32
	rng1       uint32
	dat1       uint32
	fif1       uint32
	resvd_0x1c uint32
	rng2       uint32
	dat2       uint32
}

type gpioT struct {
	fsel0      uint32 // GPIO Function Select
	fsel1      uint32
	fsel2      uint32
	fsel3      uint32
	fsel4      uint32
	fsel5      uint32
	resvd_0x18 uint32
	set0       uint32 // GPIO Pin Output Set
	set1       uint32
	resvc_0x24 uint32
	clr0       uint32 // GPIO Pin Output Clear
	clr1       uint32
	resvd_0x30 uint32
	lev0       uint32 // GPIO Pin Level
	lev1       uint32
	resvd_0x3c uint32
	eds0       uint32 // GPIO Pin Event Detect Status
	eds1       uint32
	resvd_0x48 uint32
	ren0       uint32 // GPIO Pin Rising Edge Detect Enable
	ren1       uint32
	resvd_0x54 uint32
	fen0       uint32 // GPIO Pin Falling Edge Detect Enable
	fen1       uint32
	resvd_0x60 uint32
	hen0       uint32 // GPIO Pin High Detect Enable
	hen1       uint32
	resvd_0x6c uint32
	len0       uint32 // GPIO Pin Low Detect Enable
	len1       uint32
	resvd_0x78 uint32
	aren0      uint32 // GPIO Pin Async Rising Edge Detect
	aren1      uint32
	resvd_0x84 uint32
	afen0      uint32 // GPIO Pin Async Falling Edge Detect
	afen1      uint32
	resvd_0x90 uint32
	pud        uint32 // GPIO Pin Pull up/down Enable
	pudclk0    uint32 // GPIO Pin Pull up/down Enable Clock
	pudclk1    uint32
	resvd_0xa0 uint32
	resvd_0xa4 uint32
	resvd_0xa8 uint32
	resvd_0xac uint32
	test       uint32
}

type cmClkT struct {
	ctl uint32
	div uint32
}

func (ws *WS281x) mapDmaRegisters(dma int) error {
	offset, ok := dmaOffsets[dma]
	if !ok {
		return fmt.Errorf("no offset found for DMA %d", dma)
	}
	offset += ws.rp.periphBase
	var (
		err     error
		bufOffs uintptr
	)
	ws.dmaBuf, bufOffs, err = ws.mapMem(offset, int(unsafe.Sizeof(dmaT{})))
	if err != nil {
		return fmt.Errorf("couldn't map dmaT at %08X: %v", offset, err)
	}
	fmt.Printf("Got dmaBuf[%d], offset %d\n", len(ws.dmaBuf), bufOffs)
	ws.dma = (*dmaT)(unsafe.Pointer(&ws.dmaBuf[bufOffs]))

	ws.pwmBuf, bufOffs, err = ws.mapMem(PWM_OFFSET+ws.rp.periphBase, int(unsafe.Sizeof(pwmT{})))
	if err != nil {
		return fmt.Errorf("couldn't map pwmT at %08X: %v", PWM_OFFSET+ws.rp.periphBase, err)
	}
	fmt.Printf("Got pwmBuf[%d], offset %d\n", len(ws.pwmBuf), bufOffs)
	ws.pwm = (*pwmT)(unsafe.Pointer(&ws.pwmBuf[bufOffs]))

	ws.gpioBuf, bufOffs, err = ws.mapMem(GPIO_OFFSET+ws.rp.periphBase, int(unsafe.Sizeof(gpioT{})))
	if err != nil {
		return fmt.Errorf("couldn't map gpioT at %08X: %v", GPIO_OFFSET+ws.rp.periphBase, err)
	}
	fmt.Printf("Got gpioBuf[%d], offset %d\n", len(ws.gpioBuf), bufOffs)
	ws.gpio = (*gpioT)(unsafe.Pointer(&ws.gpioBuf[bufOffs]))

	ws.cmClkBuf, bufOffs, err = ws.mapMem(CM_PWM_OFFSET+ws.rp.periphBase, int(unsafe.Sizeof(cmClkT{})))
	if err != nil {
		return fmt.Errorf("couldn't map cmClkT at %08X: %v", CM_PWM_OFFSET+ws.rp.periphBase, err)
	}
	fmt.Printf("Got cmClkBuf[%d], offset %d\n", len(ws.cmClkBuf), bufOffs)
	ws.cmClk = (*cmClkT)(unsafe.Pointer(&ws.cmClkBuf[bufOffs]))

	return nil
}
