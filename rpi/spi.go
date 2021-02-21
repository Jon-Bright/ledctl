package rpi

const (
	SPI_IOC_MAGIC           = 'k'
	SPI_IOC_WR_MAX_SPEED_HZ = 4
)

func (rp *RPi) SetSPISpeed(fd uintptr, s uint32) error {
	return ioctlUint32(fd, iow(SPI_IOC_MAGIC, SPI_IOC_WR_MAX_SPEED_HZ, uintptr(0)), s)
}
