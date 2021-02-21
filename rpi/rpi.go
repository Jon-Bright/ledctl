package rpi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	mmap "github.com/edsrzf/mmap-go"
	"os"
)

type RPi struct {
	mbox     *os.File
	mboxSize uint32
	hw       *hw
	dmaBuf   mmap.MMap
	dma      *dmaT
	pwmBuf   mmap.MMap
	pwm      *pwmT
	gpioBuf  mmap.MMap
	gpio     *gpioT
	cmClkBuf mmap.MMap
	cmClk    *cmClkT
}

func NewRPi() (*RPi, error) {
	hw, err := detectHardware()
	if err != nil {
		return nil, fmt.Errorf("couldn't detect RPi hardware: %v", err)
	}
	rp := RPi{
		hw: hw,
	}
	err = rp.mboxOpen()
	if err != nil {
		return nil, fmt.Errorf("couldn't open mailbox: %v", err)
	}
	return &rp, nil
}

type hw struct {
	hwType     int
	periphBase uintptr
	vcBase     uintptr
	name       string
}

const (
	RPI_HWVER_TYPE_UNKNOWN = iota
	RPI_HWVER_TYPE_PI1
	RPI_HWVER_TYPE_PI2
	RPI_HWVER_TYPE_PI4

	PERIPH_BASE_RPI  = 0x20000000
	PERIPH_BASE_RPI2 = 0x3f000000
	PERIPH_BASE_RPI4 = 0xfe000000

	VIDEOCORE_BASE_RPI  = 0x40000000
	VIDEOCORE_BASE_RPI2 = 0xc0000000
)

// Detect which version of a Raspberry Pi we're running on
// The original rpihw.c does this in two different ways, one for ARM64 only.
// My non-64-bit RPis also support the ARM64 way, though, so this implements just that (easier) way.
func detectHardware() (*hw, error) {
	f, err := os.Open("/proc/device-tree/system/linux,revision")
	if err != nil {
		return nil, fmt.Errorf("couldn't open linux revision file: %v", err)
	}
	b := make([]byte, 4)
	n, err := f.Read(b)
	f.Close() // Ignore error
	if err != nil {
		return nil, fmt.Errorf("couldn't read revision: %v", err)
	}
	if n != 4 {
		return nil, fmt.Errorf("revision file got %d instead of 4 bytes", n)
	}
	r := bytes.NewReader(b)
	var ver uint32
	err = binary.Read(r, binary.BigEndian, &ver)
	if err != nil {
		return nil, fmt.Errorf("somehow couldn't convert 4 bytes to a uint32: %v", err)
	}
	if rp, ok := rasPiVariants[ver]; ok {
		return &rp, nil
	}
	return nil, fmt.Errorf("couldn't identify hardware revision %X", ver)
}

var rasPiVariants = map[uint32]hw{
	//
	// Raspberry Pi 400
	//
	0xC03130: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 400 - 4GB v1.0",
	},
	//
	// Raspberry Pi 4
	//
	0xA03111: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 1GB v1.1",
	},
	0xB03111: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 2GB v.1.1",
	},
	0xC03111: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 4GB v1.1",
	},
	0xA03112: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 1GB v1.2",
	},
	0xB03112: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 2GB v.1.2",
	},
	0xC03112: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 4GB v1.2",
	},
	0xD03114: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 8GB v1.2",
	},
	0xB03114: {
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 4 Model B - 2GB v1.4",
	},
	//
	// Model B Rev 1.0
	//
	0x02: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},
	0x03: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},

	//
	// Model B Rev 2.0
	//
	0x04: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},
	0x05: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},
	0x06: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},

	//
	// Model A
	//
	0x07: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model A",
	},
	0x08: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model A",
	},
	0x09: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model A",
	},

	//
	// Model B
	//
	0x0d: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},
	0x0e: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},
	0x0f: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B",
	},

	//
	// Model B+
	//
	0x10: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B+",
	},
	0x13: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B+",
	},
	0x900032: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model B+",
	},

	//
	// Compute Module
	//
	0x11: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Compute Module 1",
	},
	0x14: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Compute Module 1",
	},

	//
	// Pi Zero
	//
	0x900092: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Pi Zero v1.2",
	},
	0x900093: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Pi Zero v1.3",
	},
	0x920093: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Pi Zero v1.3",
	},
	0x9200c1: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Pi Zero W v1.1",
	},
	0x9000c1: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Pi Zero W v1.1",
	},

	//
	// Model A+
	//
	0x12: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model A+",
	},
	0x15: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model A+",
	},
	0x900021: {
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Model A+",
	},

	//
	// Pi 2 Model B
	//
	0xA01041: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 2",
	},
	0xA01040: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 2",
	},
	0xA21041: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 2",
	},
	//
	// Pi 2 with BCM2837
	//
	0xA22042: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 2",
	},
	//
	// Pi 3 Model B
	//
	0xA020D3: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 3 B+",
	},
	0xA02082: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 3",
	},
	0xA02083: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 3",
	},
	0xA22082: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 3",
	},
	0xA22083: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Pi 3",
	},
	0x9020e0: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Model 3 A+",
	},

	//
	// Pi Compute Module 3
	//
	0xA020A0: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Compute Module 3/L3",
	},
	//
	// Pi Compute Module 3+
	//
	0xA02100: {
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Compute Module 3+",
	},
}
