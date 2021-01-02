package pixarray

import (
	"fmt"
	"os"
	"path"
	"syscall"
	"unsafe"
)

const (
	VIDEOCORE_MAJOR_NUM = 100
	VCIO_FILE           = "/dev/vcio"
	MBOX_DEV            = 100 << 20 // Assumes devices have 12-bit major, 20-bit minor numbers
	MBOX_MODE           = 0600
	LED_RESET_US        = 55
	RPI_PWM_CHANNELS    = 2
	PAGE_SIZE           = 4096 // Theoretically, we could get this via whatever getconf does
)

type dmaCallback struct {
	ti        uint32
	source_ad uint32
	dest_ad   uint32
	txfr_len  uint32
	stride    uint32
	nextconbk uint32
	resvd1    uint32
	resvd2    uint32
}

func (ws *WS281x) mboxOpenTemp() (*os.File, error) {
	tf := path.Join(os.TempDir(), fmt.Sprintf("mailbox-%d", os.Getpid()))
	err := os.Remove(tf)
	if err != nil && err != os.ErrNotExist {
		return nil, fmt.Errorf("couldn't remove temp mbox: %v", err)
	}
	err = syscall.Mknod(tf, syscall.S_IFCHR|MBOX_MODE, MBOX_DEV)
	if err != nil {
		return nil, fmt.Errorf("couldn't make device node: %v", err)
	}
	f, err := os.OpenFile(tf, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("couldn't open temp mbox: %v", err)
	}
	err = os.Remove(tf)
	if err != nil {
		f.Close() // Ignore error
		return nil, fmt.Errorf("couldn't remove temp mbox: %v", err)
	}
	return f, nil
}

func (ws *WS281x) mboxOpen() (*os.File, error) {
	f, err := os.OpenFile(VCIO_FILE, os.O_RDONLY, os.ModePerm)
	if err == os.ErrNotExist {
		f, err = ws.mboxOpenTemp()
	}
	if err != nil {
		return nil, fmt.Errorf("couldn't open mbox: %v", err)
	}
	return f, nil
}

func (ws *WS281x) mboxClose() error {
	return ws.mbox.Close()
}

func (ws *WS281x) mboxProperty(buf []uint32) error {
	f := ws.mbox
	if f == nil {
		var err error
		f, err = ws.mboxOpen()
		if err != nil {
			return fmt.Errorf("mbox not open, couldn't open: %v", err)
		}
	}

	mboxProperty := iowr(VIDEOCORE_MAJOR_NUM, 0, uintptr(0))
	if f != nil {
		err := ioctlArrUint32(f.Fd(), mboxProperty, buf)

		if err != nil {
			return fmt.Errorf("failed ioctl mbox property: %v", err)
		}
	}

	if ws.mbox == nil {
		err := f.Close()
		if err != nil {
			return fmt.Errorf("couldn't close mbox?!: %v", err)
		}
	}

	return nil
}

func (ws *WS281x) calcMboxSize(freq uint) {
	// Every bit transmitted needs 3 bits of buffer, because bits are transmitted as
	// ‾|__ (0) or ‾‾|_ (1). Each color of each pixel needs 8 "real" bits.
	bits := uint(ws.numPixels * ws.numColors * 8 * 3)

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
	bytes -= bytes % 8
	bytes += 8

	// Add 4 bytes for "idle low/high times"
	// TODO: WTF is this?
	bytes += 4

	bytes *= RPI_PWM_CHANNELS

	bytes += uint(unsafe.Sizeof(dmaCallback{}))

	// Our actual size is then whatever the next multiple of PAGE_SIZE is
	// NB: for anything less than ~1010 total RGBW pixels, this means all the juggling above works
	// out to 4096.
	ws.mboxSize = uint32(((bytes / PAGE_SIZE) + 1) * PAGE_SIZE)
}

func (ws *WS281x) allocMem() (uintptr, error) {
	i := uint32(0)
	p := make([]uint32, 32)
	p[i] = 0 // size
	i++
	p[i] = 0x00000000 // process request
	i++

	p[i] = 0x3000c // tag ID for "allocate memory"
	i++
	p[i] = 12 // size of the tag value to follow
	i++
	p[i] = 0 // afaict, mailbox.c has this wrong, we just need bit 31 clear, rest is reserved
	i++
	// tag value
	p[i] = ws.mboxSize // size of the block we want, in bytes
	i++
	p[i] = PAGE_SIZE // alignment - we want it aligned to a page boundary
	i++

	// Unclear why this difference: original code has no comment and the commit that added this
	// was just "Finish RPI2 changes. Ready for testing."
	if ws.rp.vcBase == VIDEOCORE_BASE_RPI {
		p[i] = 0xC // MEM_FLAG_L1_NONALLOCATING
	} else {
		p[i] = 0x4 // MEM_FLAG_DIRECT
	}
	i++
	p[i] = 0 // no more tags
	i++
	p[0] = i * 4 // actual size of the tag

	err := ws.mboxProperty(p)
	if err != nil {
		return 0, fmt.Errorf("can't allocate videocore memory: %v", err)
	}
	return uintptr(p[5]), nil
}
