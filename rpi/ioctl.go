package rpi

import (
	"reflect"
	"syscall"
	"unsafe"
)

// ioctl implementation.  This is basically a golang replica of
// https://github.com/raspberrypi/linux/blob/rpi-5.4.y/include/uapi/asm-generic/ioctl.h

const (
	_IOC_NRBITS   uint32 = 8
	_IOC_TYPEBITS uint32 = 8

	_IOC_SIZEBITS uint32 = 14
	_IOC_DIRBITS         = 2

	_IOC_NRSHIFT   = 0
	_IOC_TYPESHIFT = (_IOC_NRSHIFT + _IOC_NRBITS)
	_IOC_SIZESHIFT = (_IOC_TYPESHIFT + _IOC_TYPEBITS)
	_IOC_DIRSHIFT  = (_IOC_SIZESHIFT + _IOC_SIZEBITS)

	_IOC_NONE  = 0
	_IOC_WRITE = 1
	_IOC_READ  = 2
)

func ioc(dir uint32, typ uint32, nr uint32, size uint32) uint32 {
	return (((dir) << _IOC_DIRSHIFT) |
		((typ) << _IOC_TYPESHIFT) |
		((nr) << _IOC_NRSHIFT) |
		((size) << _IOC_SIZESHIFT))
}

func io(typ uint32, nr uint32) uint32 {
	return ioc(_IOC_NONE, typ, nr, 0)
}

func ior(typ uint32, nr uint32, size interface{}) uint32 {
	return ioc(_IOC_READ, typ, nr, uint32(reflect.TypeOf(size).Size()))
}

func iow(typ uint32, nr uint32, size interface{}) uint32 {
	return ioc(_IOC_WRITE, typ, nr, uint32(reflect.TypeOf(size).Size()))
}

func iowr(typ uint32, nr uint32, size interface{}) uint32 {
	return ioc(_IOC_READ|_IOC_WRITE, typ, nr, uint32(reflect.TypeOf(size).Size()))
}

func ioctlArrUint32(fd uintptr, ioctl uint32, val []uint32) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctl),
		uintptr(unsafe.Pointer(&val[0])),
	)
	var err error
	err = nil
	if errno != 0 {
		err = errno
	}
	return err
}

func ioctlUint32(fd uintptr, ioctl uint32, val uint32) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctl),
		uintptr(unsafe.Pointer(&val)),
	)
	var err error
	err = nil
	if errno != 0 {
		err = errno
	}
	return err
}
