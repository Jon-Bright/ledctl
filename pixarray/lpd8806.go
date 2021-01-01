package pixarray

import (
	"fmt"
	"syscall"
	"unsafe"
)

type LPD8806Array struct {
	baseArray
	dev       dev
	sendBytes []byte
}

func NewLPD8806(dev dev, numPixels int, spiSpeed uint32, order int) (PixArray, error) {
	numReset := (numPixels + 31) / 32
	val := make([]byte, numPixels*3+numReset)
	ba := newBaseArray(numPixels, val[:numPixels*3], order)
	la := LPD8806Array{*ba, dev, val}

	if spiSpeed != 0 {
		err := la.setSPISpeed(spiSpeed)
		if err != nil {
			return nil, fmt.Errorf("couldn't set SPI speed: %v", err)
		}
	}

	firstReset := make([]byte, numReset)
	_, err := dev.Write(firstReset)
	if err != nil {
		return nil, fmt.Errorf("couldn't reset: %v", err)
	}
	return &la, nil
}

const (
	_SPI_IOC_WR_MAX_SPEED_HZ = 0x40046B04
)

func (la *LPD8806Array) setSPISpeed(s uint32) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(la.dev.Fd()),
		uintptr(_SPI_IOC_WR_MAX_SPEED_HZ),
		uintptr(unsafe.Pointer(&s)),
	)
	if errno == 0 {
		return nil
	}
	return errno
}

func (la *LPD8806Array) Write() error {
	_, err := la.dev.Write(la.sendBytes)
	return err
}
