package rpi

import (
	"fmt"
	"log"
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

func (rp *RPi) gpioFunctionSet(pin int, alt int) error {
	reg := pin / 10
	offset := uint((pin % 10) * 3)
	funcs := []uint32{4, 5, 6, 7, 3, 2} // See p92 in datasheet - these are the alt functions only
	if alt >= len(funcs) {
		return fmt.Errorf("%d is an invalid alt function", alt)
	}

	rp.gpio.fsel[reg] &= ^(0x7 << offset)
	rp.gpio.fsel[reg] |= (funcs[alt]) << offset
	return nil
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
