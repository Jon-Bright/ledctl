package pixarray

import (
	"fmt"
	mmap "github.com/edsrzf/mmap-go"
	"log"
	"os"
	"path"
	"reflect"
	"syscall"
	"unsafe"
)

// Many details here are from the BCM2835 reference at
// https://www.raspberrypi.org/app/uploads/2012/02/BCM2835-ARM-Peripherals.pdf
// Their page numbers are noted below
// The mailbox that most of this file deals with is documented at
// https://github.com/raspberrypi/firmware/wiki/Mailbox-property-interface

const (
	VIDEOCORE_MAJOR_NUM = 100
	MEM_FILE            = "/dev/mem"
	VCIO_FILE           = "/dev/vcio"
	MBOX_DEV            = 100 << 20 // Assumes devices have 12-bit major, 20-bit minor numbers
	MBOX_MODE           = 0600
	LED_RESET_US        = 55
	RPI_PWM_CHANNELS    = 2
	PAGE_SIZE           = 4096 // Theoretically, we could get this via whatever getconf does
)

// mmapToUintSlice does terrible things to a []byte (in the form of an MMap) to return it as []uint32
func mmapToUintSlice(m mmap.MMap) []uint32 {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&m))
	header.Len /= 4
	header.Cap /= 4
	return *(*[]uint32)(unsafe.Pointer(&header))
}

// initDmaControl points the DMA control block at the start of our pixel buffer
func (ws *WS281x) initDmaControl() {
	ws.dmaCb = (*dmaControl)(unsafe.Pointer(&ws.pixBuf[ws.pixBufOffs]))
}

// busToPhys converts a BCM2835 bus address to a physical address
func (ws *WS281x) busToPhys(busAddr uintptr) uintptr {
	return busAddr &^ 0xC0000000 // p7
}

// mapMem opens /dev/mem and uses mmap to map a given physical address into our address space.
// Since the mapping has to start at a page boundary, the physical address is rounded down to the
// nearest page boundary. mapMem returns the mapped memory and the offset that should be used to
// access it (=physAddr%PAGE_SIZE).
func (ws *WS281x) mapMem(physAddr uintptr, size int) (mmap.MMap, uintptr, error) {
	f, err := os.OpenFile(MEM_FILE, os.O_RDWR|os.O_SYNC, os.ModePerm)
	if err != nil {
		return nil, 0, fmt.Errorf("couldn't open %s: %v", MEM_FILE, err)
	}

	pagemask := ^uintptr(PAGE_SIZE - 1)
	mapAddr := physAddr & pagemask
	size += int(physAddr - mapAddr)
	log.Printf("MapRegion(f, %d, RDWR, 0, %08X), physAddr %08X, mask %08X\n", size, int64(mapAddr), physAddr, pagemask)
	mm, err := mmap.MapRegion(f, size, mmap.RDWR, 0, int64(mapAddr))
	if err != nil {
		return nil, 0, fmt.Errorf("couldn't map region (%v, %v): %v", physAddr, size, err)
	}
	f.Close() // Ignore error

	return mm, physAddr & (PAGE_SIZE - 1), nil
}

// mboxOpenTemp creates a temporary device node for ioctl-ing with the mailbox, opens it and
// immediately removes the node once it's open. It returns the opened node.
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

// mboxOpen opens /dev/vcio for ioctl-ing with the mailbox. If that doesn't exist, it passes instead
// to mboxOpenTemp to get a temporary node. It returns the opened mailbox.
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

// mboxProperty uses ioctl to send messages via the mailbox
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

// pwmByteCount calculates the number of bytes needed to store the data for PWM to send - three
// bits per WS281x bit, plus enough bits to provide an appropriate reset time afterwards at the
// given frequency. It returns that byte count.
func (ws *WS281x) pwmByteCount(freq uint) uint {
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
	bytes -= bytes % 4
	bytes += 4

	bytes *= RPI_PWM_CHANNELS

	return bytes
}

func (ws *WS281x) calcMboxSize(freq uint) {
	bytes := ws.pwmByteCount(freq)

	bytes += uint(unsafe.Sizeof(dmaControl{}))

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
	p[i] = 0 // bit 31 cleared, rest is reserved
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
		return 0, fmt.Errorf("mboxProperty failed: %v", err)
	}
	if p[4]&0x80000000 == 0 {
		return 0, fmt.Errorf("response tag unset: %v", p[4])
	}
	if p[5] == 0 {
		return 0, fmt.Errorf("out of memory")
	}
	return uintptr(p[5]), nil // 5 is the same place as mboxSize above - first part of the tag value
}

func (ws *WS281x) freeMem(handle uintptr) error {
	i := uint32(0)
	p := make([]uint32, 32)
	p[i] = 0 // size
	i++
	p[i] = 0x00000000 // process request
	i++

	p[i] = 0x3000f // tag ID for "free memory"
	i++
	p[i] = 4 // size of the tag value to follow
	i++
	p[i] = 0 // bit 31 cleared, rest is reserved
	i++

	// tag value
	p[i] = uint32(handle) // handle of the allocated memory
	i++

	p[i] = 0 // no more tags
	i++
	p[0] = i * 4 // actual size of the tag

	err := ws.mboxProperty(p)
	if err != nil {
		return fmt.Errorf("mboxProperty failed: %v", err)
	}
	if p[4]&0x80000000 == 0 {
		return fmt.Errorf("response tag unset: %v", p[4])
	}
	if p[5] != 0 {
		return fmt.Errorf("status non-zero: %v", p[5])
	}
	return nil
}

func (ws *WS281x) lockMem(handle uintptr) (uintptr, error) {
	i := uint32(0)
	p := make([]uint32, 32)
	p[i] = 0 // size
	i++
	p[i] = 0x00000000 // process request
	i++

	p[i] = 0x3000d // tag ID for "lock memory"
	i++
	p[i] = 4 // size of the tag value to follow
	i++
	p[i] = 0 // afaict, mailbox.c has this wrong, we just need bit 31 clear, rest is reserved
	i++

	// tag value
	p[i] = uint32(handle) // handle of the block we want to lock
	i++

	p[i] = 0 // no more tags
	i++

	p[0] = i * 4 // actual size of the tag

	err := ws.mboxProperty(p)
	if err != nil {
		return 0, fmt.Errorf("mboxProperty failed: %v", err)
	}
	if p[4]&0x80000000 == 0 {
		return 0, fmt.Errorf("response tag unset: %v", p[4])
	}
	return uintptr(p[5]), nil // 5 is the same place as handle above - first part of the tag value
}

func (ws *WS281x) unlockMem(handle uintptr) error {
	i := uint32(0)
	p := make([]uint32, 32)
	p[i] = 0 // size
	i++
	p[i] = 0x00000000 // process request
	i++

	p[i] = 0x3000e // tag ID for "unlock memory"
	i++
	p[i] = 4 // size of the tag value to follow
	i++
	p[i] = 0 // afaict, mailbox.c has this wrong, we just need bit 31 clear, rest is reserved
	i++

	// tag value
	p[i] = uint32(handle) // handle of the block we want to unlock
	i++

	p[i] = 0 // no more tags
	i++

	p[0] = i * 4 // actual size of the tag

	err := ws.mboxProperty(p)
	if err != nil {
		return fmt.Errorf("mboxProperty failed: %v", err)
	}
	if p[4]&0x80000000 == 0 {
		return fmt.Errorf("response tag unset: %v", p[4])
	}
	if p[5] != 0 {
		return fmt.Errorf("status non-zero: %v", p[5])
	}
	return nil
}
