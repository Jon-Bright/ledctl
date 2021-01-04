package effects

import (
	"fmt"
	pixarray "github.com/Jon-Bright/ledctl/pixarray"
	"log"
	"math"
	"time"
)

type Effect interface {
	Start(pa *pixarray.PixArray, now time.Time)
	NextStep(pa *pixarray.PixArray, now time.Time) time.Duration
	Name() string
}

func abs(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

func round(f float64) int {
	if f < 0 {
		return int(f - 0.5)
	}
	return int(f + 0.5)
}

func maxP(p pixarray.Pixel) int {
	if p.R < p.G {
		if p.G < p.B {
			if p.B < p.W {
				return p.W
			}
			return p.B
		}
		if p.G < p.W {
			return p.W
		}
		return p.G
	}
	if p.R < p.B {
		if p.B < p.W {
			return p.W
		}
		return p.B
	}
	if p.R < p.W {
		return p.W
	}
	return p.R
}

func max(a, b, c float64) float64 {
	if a < b {
		if b < c {
			return c
		}
		return b
	}
	if a < c {
		return c
	}
	return a
}

func min(a, b, c float64) float64 {
	if a > b {
		if b > c {
			return c
		}
		return b
	}
	if a > c {
		return c
	}
	return a
}

func lcm(p pixarray.Pixel) int {
	// TODO: no white support
	if p.R == 0 {
		p.R = 1
	}
	if p.G == 0 {
		p.G = 1
	}
	if p.B == 0 {
		p.B = 1
	}
	m := p.R * p.G
	for p.G != 0 {
		t := p.R % p.G
		p.R = p.G
		p.G = t
	}
	p.G = m / p.R
	m = p.G * p.B
	for p.B != 0 {
		t := p.G % p.B
		p.G = p.B
		p.B = t
	}
	return m / p.G
}

type Fade struct {
	fadeTime time.Duration
	dest     pixarray.Pixel
	startPix []pixarray.Pixel
	diffs    []pixarray.Pixel
	allSame  bool
	timeStep time.Duration
	start    time.Time
}

func NewFade(fadeTime time.Duration, dest pixarray.Pixel) *Fade {
	f := Fade{}
	f.fadeTime = fadeTime
	f.dest = dest
	return &f
}

func (f *Fade) Start(pa *pixarray.PixArray, now time.Time) {
	log.Printf("Starting Fade, dest %v", f.dest)
	var lastp pixarray.Pixel
	f.allSame = true
	f.startPix = pa.GetPixels()
	f.diffs = make([]pixarray.Pixel, pa.NumPixels())
	var maxdiff pixarray.Pixel
	for i, v := range f.startPix {
		f.diffs[i].R = f.dest.R - v.R
		f.diffs[i].G = f.dest.G - v.G
		f.diffs[i].B = f.dest.B - v.B
		f.diffs[i].W = f.dest.W - v.W
		if abs(f.diffs[i].R) > maxdiff.R {
			maxdiff.R = abs(f.diffs[i].R)
		}
		if abs(f.diffs[i].G) > maxdiff.G {
			maxdiff.G = abs(f.diffs[i].G)
		}
		if abs(f.diffs[i].B) > maxdiff.B {
			maxdiff.B = abs(f.diffs[i].B)
		}
		if abs(f.diffs[i].W) > maxdiff.W {
			maxdiff.W = abs(f.diffs[i].W)
		}
		if i > 0 && lastp != v {
			f.allSame = false
		}
		lastp = v
	}
	if maxdiff.R == 0 && maxdiff.G == 0 && maxdiff.B == 0 && maxdiff.W == 0 {
		f.timeStep = f.fadeTime
	} else {
		nsStep := f.fadeTime.Nanoseconds() / int64(lcm(maxdiff))
		if f.allSame {
			log.Printf("Starting all-same")
			nsStep = nsStep / int64(pa.NumPixels())
		}
		f.timeStep = time.Duration(nsStep)
	}
	log.Printf("Fade md.R %d, md.G %d, md.B %d, md.W %d, timestep %v", maxdiff.R, maxdiff.G, maxdiff.B, maxdiff.W, f.timeStep)
	f.start = now
}

func (f *Fade) NextStep(pa *pixarray.PixArray, now time.Time) time.Duration {
	td := now.Sub(f.start)
	pct := float64(td.Nanoseconds()) / float64(f.fadeTime.Nanoseconds())
	if pct >= 1.0 {
		// We're done
		pa.SetAll(f.dest)
		return 0
	}
	if f.allSame {
		var this, next pixarray.Pixel
		var trp, tgp, tbp, twp float64
		var nrp, ngp, nbp, nwp float64
		this.R = int(f.startPix[0].R) + int(float64(f.diffs[0].R)*pct)
		this.G = int(f.startPix[0].G) + int(float64(f.diffs[0].G)*pct)
		this.B = int(f.startPix[0].B) + int(float64(f.diffs[0].B)*pct)
		this.W = int(f.startPix[0].W) + int(float64(f.diffs[0].W)*pct)
		if f.diffs[0].R != 0 {
			trp = float64(this.R-f.startPix[0].R) / float64(f.diffs[0].R)
		} else {
			trp = 0.0
		}
		if f.diffs[0].G != 0 {
			tgp = float64(this.G-f.startPix[0].G) / float64(f.diffs[0].G)
		} else {
			tgp = 0.0
		}
		if f.diffs[0].B != 0 {
			tbp = float64(this.B-f.startPix[0].B) / float64(f.diffs[0].B)
		} else {
			tbp = 0.0
		}
		if f.diffs[0].W != 0 {
			twp = float64(this.W-f.startPix[0].W) / float64(f.diffs[0].W)
		} else {
			twp = 0.0
		}
		if f.diffs[0].R > 0 {
			nrp = float64(this.R+1-f.startPix[0].R) / float64(f.diffs[0].R)
		} else if f.diffs[0].R < 0 {
			nrp = float64(this.R-1-f.startPix[0].R) / float64(f.diffs[0].R)
		} else {
			nrp = 1.1
		}
		if f.diffs[0].G > 0 {
			ngp = float64(this.G+1-f.startPix[0].G) / float64(f.diffs[0].G)
		} else if f.diffs[0].G < 0 {
			ngp = float64(this.G-1-f.startPix[0].G) / float64(f.diffs[0].G)
		} else {
			ngp = 1.1
		}
		if f.diffs[0].B > 0 {
			nbp = float64(this.B+1-f.startPix[0].B) / float64(f.diffs[0].B)
		} else if f.diffs[0].B < 0 {
			nbp = float64(this.B-1-f.startPix[0].B) / float64(f.diffs[0].B)
		} else {
			nbp = 1.1
		}
		if f.diffs[0].W > 0 {
			nwp = float64(this.W+1-f.startPix[0].W) / float64(f.diffs[0].W)
		} else if f.diffs[0].W < 0 {
			nwp = float64(this.W-1-f.startPix[0].W) / float64(f.diffs[0].W)
		} else {
			nwp = 1.1
		}
		if nrp+ngp+nbp+nwp > 4.35 {
			log.Printf("Weird, no diffs for r,g,b,w")
			return f.timeStep
		}
		pctThroughStepR := (pct - trp) / (nrp - trp)
		pctThroughStepG := (pct - tgp) / (ngp - tgp)
		pctThroughStepB := (pct - tbp) / (nbp - tbp)
		pctThroughStepW := (pct - twp) / (nwp - twp)
		next = this
		if f.diffs[0].R != 0 {
			next.R = next.R + (f.diffs[0].R / abs(f.diffs[0].R))
		}
		if f.diffs[0].G != 0 {
			next.G = next.G + (f.diffs[0].G / abs(f.diffs[0].G))
		}
		if f.diffs[0].B != 0 {
			next.B = next.B + (f.diffs[0].B / abs(f.diffs[0].B))
		}
		if f.diffs[0].W != 0 {
			next.W = next.W + (f.diffs[0].W / abs(f.diffs[0].W))
		}
		np := float64(pa.NumPixels())
		num := pixarray.Pixel{
			int(np * pctThroughStepR),
			int(np * pctThroughStepG),
			int(np * pctThroughStepB),
			int(np * pctThroughStepW),
		}
		pa.SetPerChanAlternate(num, pa.NumPixels(), this, next)
		return f.timeStep
	}

	var p pixarray.Pixel
	var lastp pixarray.Pixel
	for i, v := range f.startPix {
		p.R = int(v.R) + int(float64(f.diffs[i].R)*pct)
		p.G = int(v.G) + int(float64(f.diffs[i].G)*pct)
		p.B = int(v.B) + int(float64(f.diffs[i].B)*pct)
		p.W = int(v.W) + int(float64(f.diffs[i].W)*pct)
		pa.SetOne(i, p)
		if i == 0 {
			// Our first time through, we have no last pixel
			f.allSame = true
		} else if lastp != p {
			f.allSame = false
		}
		lastp = p
	}
	if f.allSame {
		log.Printf("Setting all-same")
		f.timeStep = time.Duration(int64(f.timeStep) / int64(pa.NumPixels()))
	}
	return f.timeStep
}

func (f *Fade) Name() string {
	return "FADE"
}

type Rainbow struct {
	cycleTime time.Duration
	start     time.Time
}

func NewRainbow(cycleTime time.Duration) *Rainbow {
	r := Rainbow{}
	r.cycleTime = cycleTime
	return &r
}

func (r *Rainbow) Start(pa *pixarray.PixArray, now time.Time) {
	log.Printf("Starting Rainbow")
	r.start = now
}

func fToPix(f float64, o float64) int {
	f -= o
	if f < 0.0 {
		f += 1.0
	}
	if f < 0.166667 {
		return 127
	}
	if f < 0.333334 {
		return 127 - round(127*((f-0.166667)/0.166667))
	}
	if f > 0.833333 {
		return round(127 * ((f - 0.833333) / 0.166667))
	}
	return 0
}

func (r *Rainbow) NextStep(pa *pixarray.PixArray, now time.Time) time.Duration {
	pos := float64(now.Sub(r.start).Nanoseconds()) / float64(r.cycleTime.Nanoseconds())
	pos -= math.Floor(pos)
	offs := round(float64(pa.NumPixels()) * pos)

	for i := 0; i < pa.NumPixels(); i++ {
		var p pixarray.Pixel
		f := float64(i) / float64(pa.NumPixels())
		p.R = fToPix(f, 0.0)
		p.G = fToPix(f, 0.333334)
		p.B = fToPix(f, 0.666667)
		pa.SetOne((i+offs)%pa.NumPixels(), p)
	}
	return r.cycleTime / time.Duration(768)
}

func (r *Rainbow) Name() string {
	return "RAINBOW"
}

// A full cycle goes across 128*6=768 steps: R->R+G->G->G+B->B->B+R->R with each of the six arrows
// representing the 128 increments between 127 and 0 inclusive of the relevant increase or decrease.
type Cycle struct {
	cycleTime time.Duration
	fadeTime  time.Duration
	start     time.Time
	last      pixarray.Pixel
	fade      *Fade
}

func NewCycle(cycleTime time.Duration) *Cycle {
	c := Cycle{}
	c.cycleTime = cycleTime
	c.fadeTime = cycleTime / time.Duration(768)
	return &c
}

func (c *Cycle) Start(pa *pixarray.PixArray, now time.Time) {
	log.Printf("Starting Cycle")
	c.start = now
	p := pa.GetPixel(0)
	c.last = p
	m := maxP(c.last)
	switch m {
	case 0:
		// Black, let's fade to red
		log.Printf("Black->Red")
		c.last.R = 127
	case c.last.R:
		// Red max
		c.last.R = 127
		if c.last.G > c.last.B {
			log.Printf("Red->Red+Green")
			c.last.B = 0
		} else {
			log.Printf("Blue+Red->Red")
			c.last.G = 0
		}
	case c.last.G:
		// Green max
		c.last.G = 127
		if c.last.B > c.last.R {
			log.Printf("Green->Green+Blue")
			c.last.R = 0
		} else {
			log.Printf("Red+Green->Green")
			c.last.B = 0
		}
	case c.last.B:
		// Blue max
		c.last.B = 127
		if c.last.G > c.last.R {
			log.Printf("Green+Blue->Blue")
			c.last.R = 0
		} else {
			log.Printf("Blue->Blue+Red")
			c.last.G = 0
		}
	default:
		panic("One of the three colours must ==m")
	}

	if c.last != p {
		p.R -= c.last.R
		p.G -= c.last.G
		p.B -= c.last.B
		p.R = abs(p.R)
		p.G = abs(p.G)
		p.B = abs(p.B)
		m = maxP(p)
		t := c.fadeTime * time.Duration(m)
		log.Printf("First fade to %v, max dist %d -> time %s", c.last, m, t)
		c.fade = NewFade(t, c.last)
		c.fade.Start(pa, now)
	} else {
		log.Printf("Already in-cycle, no initial fade needed")
		c.NextStep(pa, now)
	}
}

func (c *Cycle) NextStep(pa *pixarray.PixArray, now time.Time) time.Duration {
	if c.fade != nil {
		t := c.fade.NextStep(pa, now)
		if t != 0 {
			// This fade will continue
			return t
		}
	}
	// Time for a new fade
	if c.last.R == 127 {
		if c.last.B > 0 {
			c.last.B--
		} else if c.last.G == 127 {
			c.last.R--
		} else {
			c.last.G++
		}
	} else if c.last.G == 127 {
		if c.last.R > 0 {
			c.last.R--
		} else if c.last.B == 127 {
			c.last.G--
		} else {
			c.last.B++
		}
	} else if c.last.B == 127 {
		if c.last.G > 0 {
			c.last.G--
		} else if c.last.R == 127 {
			c.last.B--
		} else {
			c.last.R++
		}
	} else {
		panic(fmt.Sprintf("Broken colour %v", c.last))
	}
	c.fade = NewFade(c.fadeTime, c.last)
	c.fade.Start(pa, now)
	return c.cycleTime / time.Duration(768*pa.NumPixels())
}

func (f *Cycle) Name() string {
	return "CYCLE"
}

type Zip struct {
	zipTime time.Duration
	dest    pixarray.Pixel
	start   time.Time
	lastSet int
}

func NewZip(zipTime time.Duration, dest pixarray.Pixel) *Zip {
	z := Zip{}
	z.zipTime = zipTime
	z.dest = dest
	z.lastSet = -1
	return &z
}

func (z *Zip) Start(pa *pixarray.PixArray, now time.Time) {
	log.Printf("Starting Zip")
	z.start = now
}

func (z *Zip) NextStep(pa *pixarray.PixArray, now time.Time) time.Duration {
	p := int((float64(now.Sub(z.start).Nanoseconds()) / float64(z.zipTime.Nanoseconds())) * float64(pa.NumPixels()))
	for i := z.lastSet + 1; i < pa.NumPixels() && i <= p; i++ {
		pa.SetOne(i, z.dest)
	}
	if p >= pa.NumPixels() {
		return 0
	}
	return time.Duration(z.zipTime.Nanoseconds() / int64(pa.NumPixels()))
}

func (f *Zip) Name() string {
	return "ZIP"
}

type KnightRider struct {
	pulseTime time.Duration
	pulseLen  int
	start     time.Time
}

func NewKnightRider(pulseTime time.Duration, pulseLen int) *KnightRider {
	kr := KnightRider{}
	kr.pulseTime = pulseTime
	kr.pulseLen = pulseLen
	return &kr
}

func (kr *KnightRider) Start(pa *pixarray.PixArray, now time.Time) {
	log.Printf("Starting KnightRider")
	kr.start = now
	pa.SetAll(pixarray.Pixel{0, 0, 0, 0})
}

func (kr *KnightRider) NextStep(pa *pixarray.PixArray, now time.Time) time.Duration {
	pulse := now.Sub(kr.start).Nanoseconds() / kr.pulseTime.Nanoseconds()
	pulseProgress := float64(now.Sub(kr.start).Nanoseconds()-(pulse*kr.pulseTime.Nanoseconds())) / float64(kr.pulseTime.Nanoseconds())
	pulseHead := int(float64(pa.NumPixels()+kr.pulseLen) * pulseProgress)
	pulseDir := 0
	if pulse%2 == 0 {
		pulseDir = 1
	} else {
		pulseDir = -1
		pulseHead = pa.NumPixels() - pulseHead
	}
	pulseTail := pulseHead + (pulseDir * kr.pulseLen * -1)
	if pulseTail < 0 {
		pulseTail = 0
	} else if pulseTail >= pa.NumPixels() {
		pulseTail = pa.NumPixels() - 1
	}
	rangeHead := 0
	if pulseHead < 0 {
		rangeHead = 0
	} else if pulseHead >= pa.NumPixels() {
		rangeHead = pa.NumPixels() - 1
	} else {
		rangeHead = pulseHead
	}
	for i := pulseTail; i != rangeHead; i = i + pulseDir {
		v := int((float64(kr.pulseLen-abs(pulseHead-i))/float64(kr.pulseLen))*126.0) + 1
		pa.SetOne(i, pixarray.Pixel{v, 0, 0, 0})
	}
	return time.Millisecond
}

func (f *KnightRider) Name() string {
	return "KNIGHTRIDER"
}
