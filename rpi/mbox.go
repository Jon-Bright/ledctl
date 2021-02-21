package rpi

import (
	"errors"
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
	RPI_PWM_CHANNELS    = 2
)

type PhysBuf struct {
	handle  uintptr
	busAddr uintptr
	buf     mmap.MMap
	offs    uintptr
}

// uint32Slice does terrible things to an MMap (which is itself a []byte), to returns the physical buffer
// as a []uint32. It takes care of the offset between the page boundary (where MMaps always start) and the actual
// desired mapped area and also adds any bytes specified by offs.
func (pb *PhysBuf) uint32Slice(offs uintptr) []uint32 {
	offs += pb.offs
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&pb.buf))
	header.Len -= int(offs)
	header.Len /= 4
	header.Cap -= int(offs)
	header.Cap /= 4
	header.Data += offs
	return *(*[]uint32)(unsafe.Pointer(&header))
}

func (rp *RPi) FreePhysBuf(pb *PhysBuf) error {
	var err, te error
	if pb.buf != nil {
		err = pb.buf.Unmap()
		pb.buf = nil
		// Ignore error, return it later
	}
	if pb.busAddr != 0 {
		pb.busAddr = 0
		te = rp.unlockVCMem(pb.handle)
		if err == nil {
			err = te
		}
	}
	if pb.handle != 0 {
		te = rp.freeVCMem(pb.handle)
		pb.handle = 0
		if err == nil {
			err = te
		}
	}
	return err
}

// getPhysBuf gets a buffer of Videocore memory that can be used for DMA or other purposes.
func (rp *RPi) getPhysBuf(size uint32) (*PhysBuf, error) {
	pb := PhysBuf{}
	var err error
	pb.handle, err = rp.allocVCMem(size)
	if err != nil {
		return nil, fmt.Errorf("couldn't allocMem of size %v: %v", size, err)
	}
	pb.busAddr, err = rp.lockVCMem(pb.handle)
	if err != nil {
		rp.freeVCMem(pb.handle) // Ignore error
		return nil, fmt.Errorf("couldn't lockMem(%X) of size %v: %v", pb.handle, size, err)
	}
	pb.buf, pb.offs, err = rp.mapMem(busToPhys(pb.busAddr), int(size))
	if err != nil {
		rp.unlockVCMem(pb.handle) // Ignore error
		rp.freeVCMem(pb.handle)   // Ignore error
		return nil, fmt.Errorf("couldn't map busAddr(%X) of size %v: %v", pb.busAddr, size, err)
	}
	log.Printf("mapped %d bytes, busaddr %08X, offset %d\n", size, pb.busAddr, pb.offs)
	return &pb, nil
}

// busToPhys converts a BCM2835 bus address to a physical address
func busToPhys(busAddr uintptr) uintptr {
	return busAddr &^ 0xC0000000 // p7
}

// mapMem opens /dev/mem and uses mmap to map a given physical address into our address space.
// Since the mapping has to start at a page boundary, the physical address is rounded down to the
// nearest page boundary. mapMem returns the mapped memory and the offset that should be used to
// access it (=physAddr%PAGE_SIZE).
func (rp *RPi) mapMem(physAddr uintptr, size int) (mmap.MMap, uintptr, error) {
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
func (rp *RPi) mboxOpenTemp() error {
	tf := path.Join(os.TempDir(), fmt.Sprintf("mailbox-%d", os.Getpid()))
	err := os.Remove(tf)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("couldn't remove temp mbox: %v", err)
	}
	err = syscall.Mknod(tf, syscall.S_IFCHR|MBOX_MODE, MBOX_DEV)
	if err != nil {
		return fmt.Errorf("couldn't make device node: %v", err)
	}
	f, err := os.OpenFile(tf, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("couldn't open temp mbox: %v", err)
	}
	err = os.Remove(tf)
	if err != nil {
		f.Close() // Ignore error
		return fmt.Errorf("couldn't remove temp mbox: %v", err)
	}
	rp.mbox = f
	return nil
}

// mboxOpen opens /dev/vcio for ioctl-ing with the mailbox. If that doesn't exist, it passes instead
// to mboxOpenTemp to get a temporary node. It returns the opened mailbox.
func (rp *RPi) mboxOpen() error {
	var err error
	rp.mbox, err = os.OpenFile(VCIO_FILE, os.O_RDONLY, os.ModePerm)
	if err == os.ErrNotExist {
		err = rp.mboxOpenTemp()
	}
	if err != nil {
		return fmt.Errorf("couldn't open mbox: %v", err)
	}
	return nil
}

func (rp *RPi) mboxClose() error {
	return rp.mbox.Close()
}

// mboxProperty uses ioctl to send messages via the mailbox
func (rp *RPi) mboxProperty(buf []uint32) error {
	if rp.mbox == nil {
		return errors.New("mailbox not open")
	}
	mboxProperty := iowr(VIDEOCORE_MAJOR_NUM, 0, uintptr(0))
	err := ioctlArrUint32(rp.mbox.Fd(), mboxProperty, buf)
	if err != nil {
		return fmt.Errorf("failed ioctl mbox property: %v", err)
	}
	return nil
}

func (rp *RPi) allocVCMem(size uint32) (uintptr, error) {
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
	p[i] = size // size of the block we want, in bytes
	i++
	p[i] = PAGE_SIZE // alignment - we want it aligned to a page boundary
	i++

	// Unclear why this difference: original code has no comment and the commit that added this
	// was just "Finish RPI2 changes. Ready for testing."
	if rp.hw.vcBase == VIDEOCORE_BASE_RPI {
		p[i] = 0xC // MEM_FLAG_L1_NONALLOCATING
	} else {
		p[i] = 0x4 // MEM_FLAG_DIRECT
	}
	i++
	p[i] = 0 // no more tags
	i++
	p[0] = i * 4 // actual size of the tag

	err := rp.mboxProperty(p)
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

func (rp *RPi) freeVCMem(handle uintptr) error {
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

	err := rp.mboxProperty(p)
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

func (rp *RPi) lockVCMem(handle uintptr) (uintptr, error) {
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

	err := rp.mboxProperty(p)
	if err != nil {
		return 0, fmt.Errorf("mboxProperty failed: %v", err)
	}
	if p[4]&0x80000000 == 0 {
		return 0, fmt.Errorf("response tag unset: %v", p[4])
	}
	return uintptr(p[5]), nil // 5 is the same place as handle above - first part of the tag value
}

func (rp *RPi) unlockVCMem(handle uintptr) error {
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

	err := rp.mboxProperty(p)
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
