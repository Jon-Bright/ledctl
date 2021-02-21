package rpi

import (
	"fmt"
	"log"
	"time"
	"unsafe"
)

type pwmPin struct {
	channel int
	pin     int
}

// Mapping of PWM channel/pin numbers to which "alt" function means "PWM". See p102 of datasheet.
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

func cmClkDivI(val uint32) uint32 {
	return (val & 0xfff) << 12
}

func rpiPwmDmacPanic(val uint32) uint32 {
	return (val & 0xff) << 8
}

func rpiPwmDmacDreq(val uint32) uint32 {
	return (val & 0xff) << 0
}

func rpiDmaTiPerMap(val uint32) uint32 {
	return (val & 0x1f) << 16
}

func (rp *RPi) InitPWM(freq uint, buf *DMABuf, bytes uint, pins []int) error {
	oscFreq := uint32(OSC_FREQ)
	if rp.hw.hwType == RPI_HWVER_TYPE_PI4 {
		oscFreq = OSC_FREQ_PI4
	}

	for channel, pin := range pins {
		alt, ok := pwmPinToAlt[pwmPin{channel, pin}]
		if !ok {
			return fmt.Errorf("invalid pin %d for PWM channel %d", pin, channel)
		}
		rp.gpioFunctionSet(pin, alt)
	}

	if rp.pwmBuf == nil {
		var (
			bufOffs uintptr
			err     error
		)
		rp.pwmBuf, bufOffs, err = rp.mapMem(PWM_OFFSET+rp.hw.periphBase, int(unsafe.Sizeof(pwmT{})))
		if err != nil {
			return fmt.Errorf("couldn't map pwmT at %08X: %v", PWM_OFFSET+rp.hw.periphBase, err)
		}
		log.Printf("Got pwmBuf[%d], offset %d\n", len(rp.pwmBuf), bufOffs)
		rp.pwm = (*pwmT)(unsafe.Pointer(&rp.pwmBuf[bufOffs]))

		// This could potentially be in a clk.go. Seems not worth it yet, though.
		rp.cmClkBuf, bufOffs, err = rp.mapMem(CM_PWM_OFFSET+rp.hw.periphBase, int(unsafe.Sizeof(cmClkT{})))
		if err != nil {
			return fmt.Errorf("couldn't map cmClkT at %08X: %v", CM_PWM_OFFSET+rp.hw.periphBase, err)
		}
		log.Printf("Got cmClkBuf[%d], offset %d\n", len(rp.cmClkBuf), bufOffs)
		rp.cmClk = (*cmClkT)(unsafe.Pointer(&rp.cmClkBuf[bufOffs]))
	}

	rp.StopPWM()

	// Set up the clock - Use OSC @ 19.2Mhz w/ 3 clocks/tick
	rp.cmClk.div = CM_CLK_DIV_PASSWD | cmClkDivI(oscFreq/(3*uint32(freq)))
	rp.cmClk.ctl = CM_CLK_CTL_PASSWD | CM_CLK_CTL_SRC_OSC
	rp.cmClk.ctl = CM_CLK_CTL_PASSWD | CM_CLK_CTL_SRC_OSC | CM_CLK_CTL_ENAB
	time.Sleep(10 * time.Microsecond)
	log.Printf("Waiting for cmClk busy\n")
	i := 0
	for (rp.cmClk.ctl & CM_CLK_CTL_BUSY) == 0 {
		i++
	}
	log.Printf("Done %d\n", i)

	// Set up the PWM, use delays as the block is rumored to lock up without them.  Make
	// sure to use a high enough priority to avoid any FIFO underruns, especially if
	// the CPU is busy doing lots of memory accesses, or another DMA controller is
	// busy.  The FIFO will clock out data at a much slower rate (2.6Mhz max), so
	// the odds of a DMA priority boost are extremely low.

	rp.pwm.rng1 = 32 // 32-bits per word to serialize
	time.Sleep(10 * time.Microsecond)
	rp.pwm.ctl = RPI_PWM_CTL_CLRF1
	time.Sleep(10 * time.Microsecond)
	rp.pwm.dmac = RPI_PWM_DMAC_ENAB | rpiPwmDmacPanic(7) | rpiPwmDmacDreq(3)
	time.Sleep(10 * time.Microsecond)
	rp.pwm.ctl = RPI_PWM_CTL_USEF1 | RPI_PWM_CTL_MODE1 | RPI_PWM_CTL_USEF2 | RPI_PWM_CTL_MODE2
	time.Sleep(10 * time.Microsecond)
	rp.pwm.ctl |= RPI_PWM_CTL_PWEN1 | RPI_PWM_CTL_PWEN2

	// Initialize the DMA control block
	buf.c.ti = RPI_DMA_TI_NO_WIDE_BURSTS | // 32-bit transfers
		RPI_DMA_TI_WAIT_RESP | // wait for write complete
		RPI_DMA_TI_DEST_DREQ | // user peripheral flow control
		rpiDmaTiPerMap(5) | // PWM peripheral
		RPI_DMA_TI_SRC_INC // Increment src addr

	buf.c.sourceAd = uint32(buf.pb.busAddr + unsafe.Sizeof(dmaControl{}))
	log.Printf("DMA sourceAd %08X\n", buf.c.sourceAd)

	buf.c.destAd = PWM_PERIPH_PHYS + uint32(unsafe.Offsetof(rp.pwm.fif1))
	buf.c.txLen = uint32(bytes)
	log.Printf("DMA txLen %d\n", buf.c.txLen)
	buf.c.stride = 0
	buf.c.nextconbk = 0

	rp.dma.cs = 0
	rp.dma.txLen = 0
	return nil
}

func (rp *RPi) StopPWM() {
	// Turn off the PWM in case already running
	rp.pwm.ctl = 0
	time.Sleep(10 * time.Microsecond)

	// Kill the clock if it was already running
	rp.cmClk.ctl = CM_CLK_CTL_PASSWD | CM_CLK_CTL_KILL
	time.Sleep(10 * time.Microsecond)
	log.Printf("Waiting for cmClk not-busy\n")
	i := 0
	for (rp.cmClk.ctl & CM_CLK_CTL_BUSY) != 0 {
		i++
	}
	log.Printf("Done %d\n", i)
}
