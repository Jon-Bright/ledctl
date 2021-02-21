package main

import (
	"flag"
	"fmt"
	rpi "github.com/Jon-Bright/ledctl/rpi"
	"log"
	"time"
)

var powerCtrlPin = flag.Int("powerCtrlPin", -1, "A GPIO pin which, when set high, turns on power for the LEDs. -1 means no such pin exists.")
var powerStatusPin = flag.Int("powerStatusPin", -1, "A GPIO pin which indicates healthy power to the LEDs. -1 means no such pin exists. Only relevant if powerCtrlPin is specified.")
var powerStatusWait = flag.Duration("powerStatusWait", 2*time.Second, "How long to wait for a healthy power signal. Only relevant if powerStatusPin is specified and relevant.")

func initPower(rp *rpi.RPi) error {
	if *powerCtrlPin < 0 {
		return nil
	}
	err := rp.GPIOSetOutput(*powerCtrlPin, rpi.PullNone)
	if err != nil {
		return fmt.Errorf("couldn't set power control to output: %v", err)
	}
	if *powerStatusPin < 0 {
		return nil
	}
	return rp.GPIOSetInput(*powerStatusPin)
}

func powerOn(rp *rpi.RPi) error {
	if *powerCtrlPin < 0 {
		return nil
	}
	log.Printf("Power on")
	err := rp.GPIOSetPin(*powerCtrlPin, true)
	if err != nil {
		return fmt.Errorf("couldn't set power control high: %v", err)
	}
	if *powerStatusPin < 0 {
		return nil
	}
	start := time.Now()
	for {
		val, err := rp.GPIOGetPin(*powerStatusPin)
		if err != nil {
			return fmt.Errorf("couldn't query power status: %v", err)
		}
		t := time.Now()
		if val {
			log.Printf("Power stablized after %v", t.Sub(start))
			return nil
		}
		if t.Sub(start) > *powerStatusWait {
			return fmt.Errorf("timed out waiting for power to be healthy, started %v, now %v", start, t)
		}
		time.Sleep(50 * time.Millisecond) // No point overdoing it - we're not in _that_ much of a rush
	}
}

func powerOff(rp *rpi.RPi) error {
	if *powerCtrlPin < 0 {
		return nil
	}
	log.Printf("Power off")
	err := rp.GPIOSetPin(*powerCtrlPin, false)
	if err != nil {
		return fmt.Errorf("couldn't set power control low: %v", err)
	}
	// We could wait for power status to go low, but that might take a while and doesn't seem to provide any benefit
	return nil
}
