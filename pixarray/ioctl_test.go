package pixarray

import (
	"testing"
)

// These tests aren't really useful for regression purposes (difficult to see how some bit
// shifts are going to break), but they were helpful to me in confirming that the implementation
// is actually correctly translated from C world.
//
// The magic "want" numbers in these test cases were produced from this C code:
//
// #include <stdio.h>
// #include <linux/ioctl.h>
// #include <linux/spi/spidev.h>
//
// #define MAJOR_NUM 100
//
// int main(void) {
//    printf("SPI_IOC_WR_BITS_PER_WORD: %08X\n", SPI_IOC_WR_BITS_PER_WORD);
//    printf("SPI_IOC_WR_MAX_SPEED_HZ: %08X\n", SPI_IOC_WR_MAX_SPEED_HZ);
//    printf("SPI_IOC_RD_BITS_PER_WORD: %08X\n", SPI_IOC_RD_BITS_PER_WORD);
//    printf("SPI_IOC_RD_MAX_SPEED_HZ: %08X\n", SPI_IOC_RD_MAX_SPEED_HZ);
//    printf("IOCTL_MBOX_PROPERTY: %08X\n", _IOWR(MAJOR_NUM, 0, char *));
// }
//
// Which produced this output:
//
// $ ./spiconst
// SPI_IOC_WR_BITS_PER_WORD: 40016B03
// SPI_IOC_WR_MAX_SPEED_HZ: 40046B04
// SPI_IOC_RD_BITS_PER_WORD: 80016B03
// SPI_IOC_RD_MAX_SPEED_HZ: 80046B04
// IOCTL_MBOX_PROPERTY: C0046400
//
// The test cases themselves are copies of the _IOW / _IOR uses in
// https://github.com/raspberrypi/linux/blob/rpi-5.4.y/include/uapi/linux/spi/spidev.h#L125

const (
	SPI_IOC_MAGIC = 'k'
	MAJOR_NUM     = 100
)

func TestIow(t *testing.T) {
	tests := []struct {
		name string
		typ  uint32
		nr   uint32
		size interface{}
		want uint32
	}{
		{"SPI_IOC_WR_BITS_PER_WORD", SPI_IOC_MAGIC, 3, uint8(0), 0x40016B03},
		{"SPI_IOC_WR_MAX_SPEED", SPI_IOC_MAGIC, 4, uint32(0), 0x40046B04},
	}

	for _, test := range tests {
		if got := iow(test.typ, test.nr, test.size); got != test.want {
			t.Errorf("iow, %s got: %08X, want: %08X", test.name, got, test.want)
		}
	}
}

func TestIor(t *testing.T) {
	tests := []struct {
		name string
		typ  uint32
		nr   uint32
		size interface{}
		want uint32
	}{
		{"SPI_IOC_RD_BITS_PER_WORD", SPI_IOC_MAGIC, 3, uint8(0), 0x80016B03},
		{"SPI_IOC_RD_MAX_SPEED", SPI_IOC_MAGIC, 4, uint32(0), 0x80046B04},
	}

	for _, test := range tests {
		if got := ior(test.typ, test.nr, test.size); got != test.want {
			t.Errorf("ior, %s got: %08X, want: %08X", test.name, got, test.want)
		}
	}
}

func TestIowr(t *testing.T) {
	tests := []struct {
		name string
		typ  uint32
		nr   uint32
		size interface{}
		want uint32
	}{
		{"IOCTL_MBOX_PROPERTY", MAJOR_NUM, 0, uintptr(0), 0xC0046400},
	}

	for _, test := range tests {
		if got := iowr(test.typ, test.nr, test.size); got != test.want {
			t.Errorf("iowr, %s got: %08X, want: %08X", test.name, got, test.want)
		}
	}
}
