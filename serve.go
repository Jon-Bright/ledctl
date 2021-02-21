package main

import (
	"bufio"
	"flag"
	"fmt"
	effects "github.com/Jon-Bright/ledctl/effects"
	pixarray "github.com/Jon-Bright/ledctl/pixarray"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var lpd8806Dev = flag.String("dev", "/dev/spidev0.0", "The SPI device on which LPD8806 LEDs are connected")
var lpd8806SpiSpeed = flag.Uint("spispeed", 1000000, "The speed to send data via SPI to LPD8806s, in Hz")
var ws281xFreq = flag.Uint("ws281xfreq", 800000, "The frequency to send data to WS2801x devices, in Hz")
var ws281xDma = flag.Int("ws281xdma", 10, "The DMA channel to use for sending data to WS281x devices")
var ws281xPin0 = flag.Int("ws281xpin0", 18, "The pin on which channel 0 should be output for WS281x devices")
var ws281xPin1 = flag.Int("ws281xpin1", 13, "The pin on which channel 1 should be output for WS281x devices")
var ledChip = flag.String("ledchip", "ws281x", "The type of LED strip to drive: one of ws281x, lpd8806")
var port = flag.Int("port", 24601, "The port that the server should listen to")
var pixels = flag.Int("pixels", 5*32, "The number of pixels to be controlled")
var pixelOrder = flag.String("order", "GRB", "The color ordering of the pixels")

type Server struct {
	pa      *pixarray.PixArray
	l       net.Listener
	c       chan effects.Effect
	laste   effects.Effect
	off     bool
	running bool
}

func NewServer(port int, pa *pixarray.PixArray) (*Server, error) {

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	c := make(chan effects.Effect)
	log.Printf("Listening on port %d", port)
	return &Server{pa, l, c, nil, true, false}, nil
}

func parseDuration(parms string) (string, time.Duration, error) {
	t := strings.SplitN(parms, " ", 2)
	d, err := time.ParseDuration(t[0] + "s")
	if err != nil {
		return "", 0, err
	}
	if len(t) == 1 {
		return "", d, nil
	}
	return t[1], d, nil
}

func (s *Server) parseColor(parms string) (string, *pixarray.Pixel, error) {
	t := strings.SplitN(parms, " ", 2)
	var p pixarray.Pixel
	n, err := fmt.Sscanf(t[0], "%02X%02X%02X%02X", &p.R, &p.G, &p.B, &p.W)
	if err != nil && err != io.EOF {
		return "", nil, err
	}
	if n != s.pa.NumColors() {
		return "", nil, fmt.Errorf("only %d tokens parsed from '%s', wanted %d", n, t[0], s.pa.NumColors())
	}
	max := s.pa.MaxPerChannel()
	if p.R > max || p.G > max || p.B > max || p.W > max {
		return "", nil, fmt.Errorf("invalid color: one or more of %d, %d, %d, %d is >%d, parsed from %s", p.R, p.G, p.B, p.W, max, t[0])
	}
	if len(t) == 1 {
		return "", &p, nil
	}
	return t[1], &p, nil
}

func (s *Server) createEffect(cmd, parms string, w *bufio.Writer) (effects.Effect, error) {
	switch {
	case cmd == "FADE_ALL":
		parms, p, err := s.parseColor(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing color: %v", err)
		}
		_, d, err := parseDuration(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration: %v", err)
		}
		return effects.NewFade(d, *p), nil
	case cmd == "ZIP_SET_ALL":
		parms, p, err := s.parseColor(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing color: %v", err)
		}
		_, d, err := parseDuration(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration: %v", err)
		}
		return effects.NewZip(d, *p), nil
	case cmd == "CYCLE":
		_, d, err := parseDuration(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration: %v", err)
		}
		return effects.NewCycle(d), nil
	case cmd == "RAINBOW":
		_, d, err := parseDuration(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration: %v", err)
		}
		return effects.NewRainbow(d), nil
	case cmd == "GET":
		for _, p := range s.pa.GetPixels() {
			if p.R != 0 || p.G != 0 || p.B != 0 {
				w.WriteString("1\n")
				err := w.Flush()
				return nil, err
			}
		}
		w.WriteString("0\n")
		err := w.Flush()
		return nil, err
	case cmd == "COLOUR" || cmd == "COLOR":
		p := s.pa.GetPixels()[0]
		c := p.String() + "\n"
		log.Printf("Returning %s", c)
		w.WriteString(c)
		err := w.Flush()
		return nil, err
	case cmd == "MODE":
		n := "CONST"
		if s.off {
			n = "OFF"
		} else if s.running {
			if s.laste == nil {
				return nil, fmt.Errorf("s running, but laste nil!")
			}
			n = s.laste.Name()
		}
		log.Printf("Mode '%s'", n)
		if parms == "" {
			log.Printf("Returning %s", n)
			w.WriteString(n + "\n")
			err := w.Flush()
			return nil, err
		}
		r := "0\n"
		if parms == n {
			r = "1\n"
		}
		log.Printf("Returning %s", r)
		w.WriteString(r)
		err := w.Flush()
		return nil, err
	case cmd == "ON":
		return s.laste, nil
	case cmd == "OFF":
		// Hack: we insert this directly into the channel because we don't want to overwrite whatever the last effect was
		fb := effects.NewFade(20*time.Second, pixarray.Pixel{R: 0, G: 0, B: 0, W: 0})
		s.off = true
		s.c <- fb
		return nil, nil
	case cmd == "KNIGHTRIDER":
		_, d, err := parseDuration(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration: %v", err)
		}
		return effects.NewKnightRider(d, s.pa.NumPixels()/4), nil
	}
	return nil, fmt.Errorf("unknown command: %s", cmd)
}

func (s *Server) runEffects() {
	var laste, e effects.Effect
	var d time.Duration
	var steps int
	var start time.Time
	for {
		if d == 0 {
			e = <-s.c
		} else {
			select {
			case e = <-s.c:
				break
			case <-time.After(d):
				break
			}
		}
		if e == nil {
			log.Fatalf("Ready to process effect, but no effect!")
		}
		if e != laste {
			err := powerOn(s.pa.RPi())
			if err != nil {
				log.Fatalf("Failed power-on: %v", err)
			}
			start = time.Now()
			e.Start(s.pa, start)
			s.running = true
			steps = 0
		}
		d = e.NextStep(s.pa, time.Now())
		steps++
		s.pa.Write()
		if d == 0 {
			d := time.Since(start)
			ps := time.Duration(d.Nanoseconds() / int64(steps))
			log.Printf("Finished effect, %d steps, %s total, %s/step", steps, d, ps)
			laste = nil
			e = nil
			s.running = false
			p := s.pa.GetPixels()[0]
			log.Printf("Seeing post-effect pix %v", p)
			if p.R <= 0 && p.G <= 0 && p.B <= 0 && p.W <= 0 {
				err := powerOff(s.pa.RPi())
				if err != nil {
					log.Fatalf("Failed power-off: %v", err)
				}
			}
		} else {
			laste = e
		}
	}
}

func (s *Server) handleConnection(c net.Conn) {
	log.Printf("Handling connection from %v", c.RemoteAddr())
	defer c.Close()
	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	for {
		l, err := r.ReadString('\n')
		if err == io.EOF {
			log.Printf("EOF for connection %v", c.RemoteAddr())
			return
		}
		if err != nil {
			log.Printf("Error reading string for connection %v: %v", c.RemoteAddr(), err)
			return
		}
		l = strings.TrimSpace(l)
		log.Printf("Got line '%s'", l)
		t := strings.SplitN(l, " ", 2)
		cmd := strings.ToUpper(t[0])
		parms := ""
		if len(t) > 1 {
			parms = t[1]
		}
		if cmd == "QUIT" {
			return
		}
		e, err := s.createEffect(cmd, parms, w)
		if err != nil {
			es := fmt.Sprintf("Error creating effect: %v", err)
			log.Print(es)
			w.WriteString("ERR: " + es + "\n")
			err = w.Flush()
			if err != nil {
				log.Printf("error writing error reply: %v", err)
			}
			return
		}
		if e != nil {
			// Some commands don't result in a new Effect, e.g. status
			// those commands write their own reply.
			w.WriteString("OK\n")
			err = w.Flush()
			if err != nil {
				log.Printf("error writing reply: %v", err)
			}
			s.c <- e
			s.laste = e
			s.off = false
		}
	}
}

func (s *Server) handleConnections() {
	for {
		conn, err := s.l.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func main() {
	flag.Parse()
	order := pixarray.StringOrders[*pixelOrder]
	var leds pixarray.LEDStrip
	var err error
	switch *ledChip {
	case "lpd8806":
		dev, err := os.OpenFile(*lpd8806Dev, os.O_RDWR, os.ModePerm)
		if err != nil {
			log.Fatalf("Failed opening SPI: %v", err)
		}
		leds, err = pixarray.NewLPD8806(dev, *pixels, 3, uint32(*lpd8806SpiSpeed), order)
		if err != nil {
			log.Fatalf("Failed creating LPD8806: %v", err)
		}
	case "ws281x":
		leds, err = pixarray.NewWS281x(*pixels, 3, order, *ws281xFreq, *ws281xDma, []int{*ws281xPin0, *ws281xPin1})
		if err != nil {
			log.Fatalf("Failed creating WS281x: %v", err)
		}
	default:
		log.Fatalf("Unrecognized LED type: %v", *ledChip)
	}
	pa := pixarray.NewPixArray(*pixels, 3, leds) // TODO: White

	s, err := NewServer(*port, pa)
	if err != nil {
		log.Fatalf("Failed creating server: %v", err)
	}

	go s.runEffects()
	s.handleConnections()
}
