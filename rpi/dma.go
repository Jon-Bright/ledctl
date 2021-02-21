package rpi

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"
	"unsafe"
)

const (
	PAGE_SIZE       = 4096 // Theoretically, we could get this via whatever getconf does
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
	RPI_DMA_CS_RESET                      = 1 << 31
	RPI_DMA_CS_WAIT_OUTSTANDING_WRITES    = 1 << 28
	RPI_DMA_CS_ERROR                      = 1 << 8
	RPI_DMA_CS_WAITING_OUTSTANDING_WRITES = 1 << 6
	RPI_DMA_CS_INT                        = 1 << 2
	RPI_DMA_CS_END                        = 1 << 1
	RPI_DMA_CS_ACTIVE                     = 1 << 0
	RPI_DMA_TI_NO_WIDE_BURSTS             = 1 << 26
	RPI_DMA_TI_SRC_INC                    = 1 << 8
	RPI_DMA_TI_DEST_DREQ                  = 1 << 6
	RPI_DMA_TI_WAIT_RESP                  = 1 << 3
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

type dmaControl struct {
	ti        uint32
	sourceAd  uint32
	destAd    uint32
	txLen     uint32
	stride    uint32
	nextconbk uint32
	resvd1    uint32
	resvd2    uint32
}

type DMABuf struct {
	pb *PhysBuf
	c  *dmaControl
}

func (rp *RPi) GetDMABuf(bytes uint) (*DMABuf, error) {
	var d DMABuf
	var err error
	d.pb, err = rp.getPhysBuf(calcDMABufSize(bytes))
	if err != nil {
		return nil, fmt.Errorf("couldn't get %d byte phyical buffer for DMA: %v", bytes, err)
	}
	d.c = (*dmaControl)(unsafe.Pointer(&d.pb.buf[d.pb.offs]))
	log.Printf("dmabuf size %d, calc %d, addr %08X\n", bytes, calcDMABufSize(bytes), uintptr(unsafe.Pointer(d.c)))
	return &d, nil
}

func (rp *RPi) FreeDMABuf(d *DMABuf) error {
	return rp.FreePhysBuf(d.pb)
}

func (d *DMABuf) Uint32Slice() []uint32 {
	return d.pb.uint32Slice(unsafe.Sizeof(dmaControl{}))
}

// calcDMABufSize calculates how many bytes should be allocated to provide a DMA buffer with the given number of
// bytes, including the dmaControl header.
func calcDMABufSize(bytes uint) uint32 {
	bytes += uint(unsafe.Sizeof(dmaControl{}))

	// Our actual size is then whatever the next multiple of PAGE_SIZE is
	return uint32(((bytes / PAGE_SIZE) + 1) * PAGE_SIZE)
}

func (rp *RPi) InitDMA(dma int) error {
	offset, ok := dmaOffsets[dma]
	if !ok {
		return fmt.Errorf("no offset found for DMA %d", dma)
	}
	offset += rp.hw.periphBase
	var (
		bufOffs uintptr
		err     error
	)
	rp.dmaBuf, bufOffs, err = rp.mapMem(offset, int(unsafe.Sizeof(dmaT{})))
	if err != nil {
		return fmt.Errorf("couldn't map dmaT at %08X: %v", offset, err)
	}
	log.Printf("Got dmaBuf[%d], offset %d\n", len(rp.dmaBuf), bufOffs)
	rp.dma = (*dmaT)(unsafe.Pointer(&rp.dmaBuf[bufOffs]))
	return nil
}

func rpiDmaCsPanicPriority(val uint32) uint32 {
	return (val & 0xf) << 20
}

func rpiDmaCsPriority(val uint32) uint32 {
	return (val & 0xf) << 16
}

func (rp *RPi) StartDMA(d *DMABuf) {
	log.Printf("DMA to do: control %v\nData:\n%s\n", d.c, hex.Dump(d.pb.buf))
	rp.dma.cs = RPI_DMA_CS_RESET
	time.Sleep(10 * time.Microsecond)

	rp.dma.cs = RPI_DMA_CS_INT | RPI_DMA_CS_END
	time.Sleep(10 * time.Microsecond)

	rp.dma.conblkAd = uint32(d.pb.busAddr)
	rp.dma.debug = 7 // clear debug error flags
	rp.dma.cs = RPI_DMA_CS_WAIT_OUTSTANDING_WRITES |
		rpiDmaCsPanicPriority(15) |
		rpiDmaCsPriority(15) |
		RPI_DMA_CS_ACTIVE
}

func (rp *RPi) WaitForDMAEnd() error {
	var cs uint32
	i := 0
	for true {
		cs = rp.dma.cs
		if (cs & RPI_DMA_CS_ACTIVE) == 0 {
			break
		}
		if (cs & RPI_DMA_CS_ERROR) != 0 {
			break
		}
		i++
		if i == 100000 {
			return fmt.Errorf("wait failed, cs %08X", cs)
		}
		time.Sleep(10 * time.Microsecond)
	}
	if (cs & RPI_DMA_CS_ERROR) != 0 {
		return fmt.Errorf("DMA error, cs %08X, debug %08X", cs, rp.dma.debug)
	}
	return nil
}
