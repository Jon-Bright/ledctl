package pixarray

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	PWM_OFFSET      = uintptr(0x0020c000)
	GPIO_OFFSET     = uintptr(0x00200000)
	CM_PWM_OFFSET   = uintptr(0x001010a0)
	PWM_PERIPH_PHYS = uint32(0x7e20c000)
	OSC_FREQ        = 19200000 // crystal frequency
	OSC_FREQ_PI4    = 54000000 // Pi 4 crystal frequency
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

const (
	RPI_DMA_TI_NO_WIDE_BURSTS = 1 << 26
	RPI_DMA_TI_SRC_INC        = 1 << 8
	RPI_DMA_TI_DEST_DREQ      = 1 << 6
	RPI_DMA_TI_WAIT_RESP      = 1 << 3
)

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

type dmaCallback struct {
	ti        uint32
	sourceAd  uint32
	destAd    uint32
	txLen     uint32
	stride    uint32
	nextconbk uint32
	resvd1    uint32
	resvd2    uint32
}

type pwmPin struct {
	channel int
	pin     int
}

var pwmPinToAlt = map[pwmPin]int{
	{0, 12}: 0,
	{0, 18}: 5,
	{0, 40}: 0,
	{1, 13}: 0,
	{1, 19}: 0,
	{1, 41}: 0,
	{1, 45}: 0,
}

const (
	RPI_PWM_CTL_USEF2 = 1 << 13
	RPI_PWM_CTL_MODE2 = 1 << 9
	RPI_PWM_CTL_PWEN2 = 1 << 8
	RPI_PWM_CTL_CLRF1 = 1 << 6
	RPI_PWM_CTL_USEF1 = 1 << 5
	RPI_PWM_CTL_MODE1 = 1 << 1
	RPI_PWM_CTL_PWEN1 = 1 << 0
	RPI_PWM_DMAC_ENAB = uint32(1 << 31)
)

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
	fsel       [6]uint32 // GPIO Function Select
	resvd_0x18 uint32
	set        [2]uint32 // GPIO Pin Output Set
	resvc_0x24 uint32
	clr        [2]uint32 // GPIO Pin Output Clear
	resvd_0x30 uint32
	lev        [2]uint32 // GPIO Pin Level
	resvd_0x3c uint32
	eds        [2]uint32 // GPIO Pin Event Detect Status
	resvd_0x48 uint32
	ren        [2]uint32 // GPIO Pin Rising Edge Detect Enable
	resvd_0x54 uint32
	fen        [2]uint32 // GPIO Pin Falling Edge Detect Enable
	resvd_0x60 uint32
	hen        [2]uint32 // GPIO Pin High Detect Enable
	resvd_0x6c uint32
	len        [2]uint32 // GPIO Pin Low Detect Enable
	resvd_0x78 uint32
	aren       [2]uint32 // GPIO Pin Async Rising Edge Detect
	resvd_0x84 uint32
	afen       [2]uint32 // GPIO Pin Async Falling Edge Detect
	resvd_0x90 uint32
	pud        uint32    // GPIO Pin Pull up/down Enable
	pudclk     [2]uint32 // GPIO Pin Pull up/down Enable Clock
	resvd_0xa0 [4]uint32
	test       uint32
}

const (
	CM_CLK_CTL_PASSWD  = 0x5a << 24
	CM_CLK_CTL_BUSY    = 1 << 7
	CM_CLK_CTL_KILL    = 1 << 5
	CM_CLK_CTL_ENAB    = 1 << 4
	CM_CLK_CTL_SRC_OSC = 1 << 0
	CM_CLK_DIV_PASSWD  = uint32(0x5a << 24)
)

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

func (ws *WS281x) gpioFunctionSet(pin int, alt int) error {
	reg := pin / 10
	offset := uint((pin % 10) * 3)
	funcs := []uint32{4, 5, 6, 7, 3, 2} // See p92 in datasheet - these are the alt functions only
	if alt >= len(funcs) {
		return fmt.Errorf("%d is an invalid alt function", alt)
	}

	ws.gpio.fsel[reg] &= ^(0x7 << offset)
	ws.gpio.fsel[reg] |= (funcs[alt]) << offset
	return nil
}

func (ws *WS281x) initGpio(pins []int) error {
	for channel, pin := range pins {
		alt, ok := pwmPinToAlt[pwmPin{channel, pin}]
		if !ok {
			return fmt.Errorf("invalid pin %d for PWM channel %d", pin, channel)
		}
		ws.gpioFunctionSet(pin, alt)
	}
	return nil
}

func (ws *WS281x) stopPwm() {
	// Turn off the PWM in case already running
	ws.pwm.ctl = 0
	time.Sleep(10 * time.Microsecond)

	// Kill the clock if it was already running
	ws.cmClk.ctl = CM_CLK_CTL_PASSWD | CM_CLK_CTL_KILL
	time.Sleep(10 * time.Microsecond)
	fmt.Printf("Waiting for cmClk not-busy\n")
	for (atomic.LoadUint32(&ws.cmClk.ctl) & CM_CLK_CTL_BUSY) != 0 {
	}
	fmt.Printf("Done\n")
}

func (ws *WS281x) cmClkDivI(val uint32) uint32 {
	return (val & 0xfff) << 12
}

func (ws *WS281x) rpiPwmDmacPanic(val uint32) uint32 {
	return (val & 0xff) << 8
}

func (ws *WS281x) rpiPwmDmacDreq(val uint32) uint32 {
	return (val & 0xff) << 0
}

func (ws *WS281x) rpiDmaTiPerMap(val uint32) uint32 {
	return (val & 0x1f) << 16
}

func (ws *WS281x) initPwm(freq uint) {
	oscFreq := uint32(OSC_FREQ)
	if ws.rp.hwType == RPI_HWVER_TYPE_PI4 {
		oscFreq = OSC_FREQ_PI4
	}

	ws.stopPwm()

	// Set up the clock - Use OSC @ 19.2Mhz w/ 3 clocks/tick
	ws.cmClk.div = CM_CLK_DIV_PASSWD | ws.cmClkDivI(oscFreq/(3*uint32(freq)))
	ws.cmClk.ctl = CM_CLK_CTL_PASSWD | CM_CLK_CTL_SRC_OSC
	ws.cmClk.ctl = CM_CLK_CTL_PASSWD | CM_CLK_CTL_SRC_OSC | CM_CLK_CTL_ENAB
	time.Sleep(10 * time.Microsecond)
	fmt.Printf("Waiting for cmClk busy\n")
	for (atomic.LoadUint32(&ws.cmClk.ctl) & CM_CLK_CTL_BUSY) == 0 {
	}
	fmt.Printf("Done\n")

	// Setup the PWM, use delays as the block is rumored to lock up without them.  Make
	// sure to use a high enough priority to avoid any FIFO underruns, especially if
	// the CPU is busy doing lots of memory accesses, or another DMA controller is
	// busy.  The FIFO will clock out data at a much slower rate (2.6Mhz max), so
	// the odds of a DMA priority boost are extremely low.

	ws.pwm.rng1 = 32 // 32-bits per word to serialize
	time.Sleep(10 * time.Microsecond)
	ws.pwm.ctl = RPI_PWM_CTL_CLRF1
	time.Sleep(10 * time.Microsecond)
	ws.pwm.dmac = RPI_PWM_DMAC_ENAB | ws.rpiPwmDmacPanic(7) | ws.rpiPwmDmacDreq(3)
	time.Sleep(10 * time.Microsecond)
	ws.pwm.ctl = RPI_PWM_CTL_USEF1 | RPI_PWM_CTL_MODE1 | RPI_PWM_CTL_USEF2 | RPI_PWM_CTL_MODE2
	time.Sleep(10 * time.Microsecond)
	ws.pwm.ctl |= RPI_PWM_CTL_PWEN1 | RPI_PWM_CTL_PWEN2

	// Initialize the DMA control block
	ws.dmaCb.ti = RPI_DMA_TI_NO_WIDE_BURSTS | // 32-bit transfers
		RPI_DMA_TI_WAIT_RESP | // wait for write complete
		RPI_DMA_TI_DEST_DREQ | // user peripheral flow control
		ws.rpiDmaTiPerMap(5) | // PWM peripheral
		RPI_DMA_TI_SRC_INC // Increment src addr

	ws.dmaCb.sourceAd = uint32(ws.pixBusAddr + unsafe.Sizeof(dmaCallback{}))

	ws.dmaCb.destAd = PWM_PERIPH_PHYS + uint32(unsafe.Offsetof(ws.pwm.fif1))
	ws.dmaCb.txLen = uint32(ws.pwmByteCount(freq))
	ws.dmaCb.stride = 0
	ws.dmaCb.nextconbk = 0

	ws.dma.cs = 0
	ws.dma.txLen = 0
}
