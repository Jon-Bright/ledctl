package rpi

const (
	CM_CLK_CTL_PASSWD  = 0x5a << 24
	CM_CLK_CTL_BUSY    = 1 << 7
	CM_CLK_CTL_KILL    = 1 << 5
	CM_CLK_CTL_ENAB    = 1 << 4
	CM_CLK_CTL_SRC_OSC = 1 << 0
	CM_CLK_DIV_PASSWD  = uint32(0x5a << 24)
)

type cmClkT struct {
	ctl uint32
	div uint32
}
