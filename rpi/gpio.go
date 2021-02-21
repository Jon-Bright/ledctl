package rpi

import (
	"fmt"
	"log"
	"time"
	"unsafe"
)

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

type PullMode uint

const (
	// See p101. These are GPPUD values
	PullNone = 0
	PullDown = 1
	PullUp   = 2
)

func (rp *RPi) gpioSetPinFunction(pin int, fnc uint32) error {
	if pin > 53 { // p94
		return fmt.Errorf("pin %d not supported")
	}
	reg := pin / 10
	offset := uint((pin % 10) * 3)
	rp.gpio.fsel[reg] &= ^(0x7 << offset)
	rp.gpio.fsel[reg] |= fnc << offset
	return nil
}

func (rp *RPi) gpioSetInput(pin int) error {
	return rp.gpioSetPinFunction(pin, 0)
}

func (rp *RPi) gpioSetOutput(pin int, pm PullMode) error {
	if pm > PullUp {
		return fmt.Errorf("%d is an invalid pull mode", pm)
	}
	err := rp.gpioSetPinFunction(pin, 1)
	if err != nil {
		return fmt.Errorf("couldn't set pin as output: %v", err)
	}

	// See p101 for the description of this procedure.
	rp.gpio.pud = uint32(pm)
	time.Sleep(10 * time.Microsecond) // Datasheet says to sleep for 150 cycles after setting pud
	reg := pin / 32
	offset := uint(pin % 32)
	rp.gpio.pudclk[reg] = 1 << offset
	time.Sleep(10 * time.Microsecond) // Datasheet says to sleep for 150 cycles after setting pudclk
	rp.gpio.pud = 0
	rp.gpio.pudclk[reg] = 0
	return nil
}

func (rp *RPi) gpioSetAltFunction(pin int, alt int) error {
	funcs := []uint32{4, 5, 6, 7, 3, 2} // See p92 in datasheet - these are the alt functions only
	if alt >= len(funcs) {
		return fmt.Errorf("%d is an invalid alt function", alt)
	}
	return rp.gpioSetPinFunction(pin, funcs[alt])
}

func (rp *RPi) InitGPIO() error {
	var (
		bufOffs uintptr
		err     error
	)
	rp.gpioBuf, bufOffs, err = rp.mapMem(GPIO_OFFSET+rp.hw.periphBase, int(unsafe.Sizeof(gpioT{})))
	if err != nil {
		return fmt.Errorf("couldn't map gpioT at %08X: %v", GPIO_OFFSET+rp.hw.periphBase, err)
	}
	log.Printf("Got gpioBuf[%d], offset %d\n", len(rp.gpioBuf), bufOffs)
	rp.gpio = (*gpioT)(unsafe.Pointer(&rp.gpioBuf[bufOffs]))
	return nil
}
