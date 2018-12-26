package main

import (
	"bufio"
	"effects"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"pixarray"
	"strings"
	"time"
)

var dev = flag.String("dev", "/dev/spidev0.0", "The SPI device on which the LEDs are connected")
var port = flag.Int("port", 24601, "The port that the server should listen to")
var pixels = flag.Int("pixels", 5*32, "The number of pixels to be controlled")

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

func parseColor(parms string) (string, *pixarray.Pixel, error) {
	t := strings.SplitN(parms, " ", 2)
	var p pixarray.Pixel
	n, err := fmt.Sscanf(t[0], "%02X%02X%02X", &p.R, &p.G, &p.B)
	if err != nil {
		return "", nil, err
	}
	if n != 3 {
		return "", nil, fmt.Errorf("only %d tokens parsed from '%s', wanted 3", n, t[0])
	}
	if p.R > 127 || p.G > 127 || p.B > 127 {
		return "", nil, fmt.Errorf("invalid color: one or more of %d, %d, %d is >127, parsed from %s", p.R, p.G, p.B, t[0])
	}
	if len(t) == 1 {
		return "", &p, nil
	}
	return t[1], &p, nil
}

func (s *Server) createEffect(cmd, parms string, w *bufio.Writer) (effects.Effect, error) {
	switch {
	case cmd == "FADE_ALL":
		parms, p, err := parseColor(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing color: %v", err)
		}
		_, d, err := parseDuration(parms)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration: %v", err)
		}
		return effects.NewFade(d, *p), nil
	case cmd == "ZIP_SET_ALL":
		parms, p, err := parseColor(parms)
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
	case cmd == "COLOUR":
		p := s.pa.GetPixels()[0]
		c := fmt.Sprintf("%02x%02x%02x\n", p.R, p.G, p.B)
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
		fb := effects.NewFade(10*time.Second, pixarray.Pixel{0, 0, 0})
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
			e.Start(s.pa)
			s.running = true
			steps = 0
		}
		d = e.NextStep(s.pa)
		steps++
		s.pa.Write()
		if d == 0 {
			log.Printf("Finished effect, %d steps", steps)
			laste = nil
			e = nil
			s.running = false
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
			log.Printf(es)
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
	dev, err := os.OpenFile(*dev, os.O_RDWR, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed opening SPI: %v", err)
	}
	pa, err := pixarray.NewPixArray(dev, *pixels)
	if err != nil {
		log.Fatalf("Failed creating PixArray: %v", err)
	}

	s, err := NewServer(*port, pa)
	if err != nil {
		log.Fatalf("Failed creating server: %v", err)
	}

	go s.runEffects()
	s.handleConnections()
}
