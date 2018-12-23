package pixarray

import "fmt"

//import "encoding/hex"
import "os"
import "syscall"
import "unsafe"

func abs(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

type Pixel struct {
	R int
	G int
	B int
}

func (p *Pixel) String() string {
	return fmt.Sprintf("%02x%02x%02x", p.R, p.G, p.B)
}

type PixArray struct {
	dev       *os.File
	numPixels int
	sendBytes []byte
	pixels    []byte
}

func NewPixArray(dev *os.File, numPixels int) (*PixArray, error) {
	numReset := (numPixels + 31) / 32
	val := make([]byte, numPixels*3+numReset)
	pa := PixArray{dev, numPixels, val, val[:numPixels*3]}

	err := pa.setSPISpeed(1000000)
	if err != nil {
		return nil, fmt.Errorf("couldn't set SPI speed: %v", err)
	}

	firstReset := make([]byte, numReset)
	_, err = dev.Write(firstReset)
	if err != nil {
		return nil, fmt.Errorf("couldn't reset: %v", err)
	}
	return &pa, nil
}

const (
	_SPI_IOC_WR_MAX_SPEED_HZ = 0x40046B04
)

func (pa *PixArray) setSPISpeed(s uint32) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(pa.dev.Fd()),
		uintptr(_SPI_IOC_WR_MAX_SPEED_HZ),
		uintptr(unsafe.Pointer(&s)),
	)
	if errno == 0 {
		return nil
	}
	return errno
}

func (pa *PixArray) Write() error {
	_, err := pa.dev.Write(pa.sendBytes)
	//fmt.Printf("Wrote %d bytes\n", len(pa.sendBytes))
	//fmt.Print(hex.Dump(pa.sendBytes))
	return err
}

func (pa *PixArray) NumPixels() int {
	return pa.numPixels
}

func (pa *PixArray) GetPixels() []Pixel {
	p := make([]Pixel, pa.numPixels)
	for i, v := range pa.pixels {
		switch i % 3 {
		case 0:
			p[i/3].G = int(v) & 0x7f
		case 1:
			p[i/3].R = int(v) & 0x7f
		case 2:
			p[i/3].B = int(v) & 0x7f
		}
	}
	return p
}

func (pa *PixArray) GetPixel(i int) Pixel {
	return Pixel{int(pa.pixels[i*3+1]) & 0x7f, int(pa.pixels[i*3]) & 0x7f, int(pa.pixels[i*3+2]) & 0x7f}
}

func (pa *PixArray) SetAlternate(num int, div int, p1 Pixel, p2 Pixel) {
	totSet := 0
	shouldSet := 0
	for i := 0; i < pa.numPixels; i++ {
		shouldSet += num
		e1 := abs((totSet + div) - shouldSet)
		e2 := abs(totSet - shouldSet)
		if e1 < e2 {
			totSet += div
			pa.pixels[i*3] = byte(0x80 | p2.G)
			pa.pixels[i*3+1] = byte(0x80 | p2.R)
			pa.pixels[i*3+2] = byte(0x80 | p2.B)
		} else {
			pa.pixels[i*3] = byte(0x80 | p1.G)
			pa.pixels[i*3+1] = byte(0x80 | p1.R)
			pa.pixels[i*3+2] = byte(0x80 | p1.B)
		}
	}
}

func (pa *PixArray) SetAll(p Pixel) {
	for i := 0; i < pa.numPixels; i++ {
		pa.pixels[i*3] = byte(0x80 | p.G)
		pa.pixels[i*3+1] = byte(0x80 | p.R)
		pa.pixels[i*3+2] = byte(0x80 | p.B)
	}
}

func (pa *PixArray) SetOne(i int, p Pixel) {
	pa.pixels[i*3] = byte(0x80 | p.G)
	pa.pixels[i*3+1] = byte(0x80 | p.R)
	pa.pixels[i*3+2] = byte(0x80 | p.B)
}
