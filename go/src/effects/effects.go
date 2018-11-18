package effects

import "fmt"
import "log"
import "pixarray"
import "time"

type Effect interface {
	Start(pa *pixarray.PixArray)
	NextStep(pa *pixarray.PixArray) time.Duration
}

func abs(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

func maxP(p pixarray.Pixel) int {
	if p.R < p.G {
		if p.G < p.B {
			return p.B
		}
		return p.G
	}
	if p.R < p.B {
		return p.B
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

func (f *Fade) Start(pa *pixarray.PixArray) {
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
		if abs(f.diffs[i].R) > maxdiff.R {
			maxdiff.R = abs(f.diffs[i].R)
		}
		if abs(f.diffs[i].G) > maxdiff.G {
			maxdiff.G = abs(f.diffs[i].G)
		}
		if abs(f.diffs[i].B) > maxdiff.B {
			maxdiff.B = abs(f.diffs[i].B)
		}
		if i > 0 && lastp != v {
			f.allSame = false
		}
		lastp = v
	}
	if maxdiff.R == 0 && maxdiff.G == 0 && maxdiff.B == 0 {
		f.timeStep = f.fadeTime
	} else {
		nsStep := f.fadeTime.Nanoseconds() / int64(lcm(maxdiff))
		if f.allSame {
			log.Printf("Starting all-same")
			nsStep = nsStep / int64(pa.NumPixels())
		}
		f.timeStep = time.Duration(nsStep)
	}
	log.Printf("Fade md.R %d, md.G %d, md.B %d, timestep %v", maxdiff.R, maxdiff.G, maxdiff.B, f.timeStep)
	f.start = time.Now()
}

func (f *Fade) NextStep(pa *pixarray.PixArray) time.Duration {
	td := time.Since(f.start)
	pct := float64(td.Nanoseconds()) / float64(f.fadeTime.Nanoseconds())
	if pct >= 1.0 {
		// We're done
		pa.SetAll(f.dest)
		return 0
	}
	if f.allSame {
		var this, next pixarray.Pixel
		var trp, tgp, tbp float64
		var nrp, ngp, nbp float64
		this.R = int(f.startPix[0].R) + int(float64(f.diffs[0].R)*pct)
		this.G = int(f.startPix[0].G) + int(float64(f.diffs[0].G)*pct)
		this.B = int(f.startPix[0].B) + int(float64(f.diffs[0].B)*pct)
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
		maxThis := max(trp, tgp, tbp)
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
		minNext := min(nrp, ngp, nbp)
		if minNext==1.1 {
			return f.timeStep
		}
		pctThroughStep := (pct - maxThis) / (minNext - maxThis)
		next = this
		if minNext == nrp {
			next.R = next.R + (f.diffs[0].R / abs(f.diffs[0].R))
		} else if minNext == ngp {
			next.G = next.G + (f.diffs[0].G / abs(f.diffs[0].G))
		} else {
			next.B = next.B + (f.diffs[0].B / abs(f.diffs[0].B))
		}
		pa.SetAlternate(int(float64(pa.NumPixels())*pctThroughStep), pa.NumPixels(), this, next)
		return f.timeStep
	}

	var p pixarray.Pixel
	var lastp pixarray.Pixel
	for i, v := range f.startPix {
		p.R = int(v.R) + int(float64(f.diffs[i].R)*pct)
		p.G = int(v.G) + int(float64(f.diffs[i].G)*pct)
		p.B = int(v.B) + int(float64(f.diffs[i].B)*pct)
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

// A full cycle goes across 128*6=768 steps: R->R+G->G->G+B->B->B+R->R with each of the six arrows
// representing the 128 increments between 127 and 0 inclusive of the relevant increase or decrease.
type Cycle struct {
	cycleTime time.Duration
	start time.Time
	last pixarray.Pixel
	fade *Fade
}

func NewCycle(cycleTime time.Duration) *Cycle {
	c := Cycle{}
	c.cycleTime = cycleTime
	return &c
}

func (c *Cycle) Start(pa *pixarray.PixArray) {
	log.Printf("Starting Cycle")
	c.start = time.Now()
	c.last = pa.GetPixel(0)
	m := maxP(c.last)
	switch m {
	case 0:
		// Black, let's fade to red
		log.Printf("Black->Red")
		c.last.R = 127
	case c.last.R:
		// Red max, let's assume we're on an R->R+G cycle
		log.Printf("Red->Red+Green")
		if c.last.G != 127 {
			c.last.R = 127
		}
		c.last.B = 0
	case c.last.G:
		// Green max, let's assume we're on a G->G+B cycle
		log.Printf("Green->Green+Blue")
		if c.last.B != 127 {
			c.last.G = 127
		}
		c.last.R = 0
	case c.last.B:
		// Blue max, let's assume we're on a B->B+R cycle
		log.Printf("Blue->Blue+Red")
		if c.last.R != 127 {
			c.last.B = 127
		}
		c.last.G = 0
	default:
		panic("One of the three colours must ==m")
	}
	log.Printf("First fade to %v", c.last)
	c.fade = NewFade(c.cycleTime / time.Duration(6), c.last)
}

func (c *Cycle) NextStep(pa *pixarray.PixArray) time.Duration {
	t := c.fade.NextStep(pa)
	if t!=0 {
		// This fade will continue
		return t
	}
	// Time for a new fade
	if c.last.R==127 {
		if c.last.B>0 {
			c.last.B--
		} else if c.last.G==127 {
			c.last.R--
		} else {
			c.last.G++
		}
	} else if c.last.G==127 {
		if c.last.R>0 {
			c.last.R--
		} else if c.last.B==127 {
			c.last.G--
		} else {
			c.last.B++
		}
	} else if c.last.B==127 {
		if c.last.G>0 {
			c.last.G--
		} else if c.last.R==127 {
			c.last.B--
		} else {
			c.last.R++
		}
	} else {
		panic(fmt.Sprintf("Broken colour %v", c.last))
	}
	c.fade = NewFade(c.cycleTime / time.Duration(786), c.last)
	c.fade.Start(pa)
	return c.cycleTime / time.Duration(786 * pa.NumPixels())
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

func (z *Zip) Start(pa *pixarray.PixArray) {
	log.Printf("Starting Zip")
	z.start = time.Now()
}

func (z *Zip) NextStep(pa *pixarray.PixArray) time.Duration {
	p := int((float64(time.Since(z.start).Nanoseconds()) / float64(z.zipTime.Nanoseconds())) * float64(pa.NumPixels()))
	for i := z.lastSet + 1; i < pa.NumPixels() && i <= p; i++ {
		pa.SetOne(i, z.dest)
	}
	if p >= pa.NumPixels() {
		return 0
	}
	return time.Duration(z.zipTime.Nanoseconds() / int64(pa.NumPixels()))
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

func (kr *KnightRider) Start(pa *pixarray.PixArray) {
	log.Printf("Starting KnightRider")
	kr.start = time.Now()
	pa.SetAll(pixarray.Pixel{0, 0, 0})
}

func (kr *KnightRider) NextStep(pa *pixarray.PixArray) time.Duration {
	pulse := time.Since(kr.start).Nanoseconds() / kr.pulseTime.Nanoseconds()
	pulseProgress := float64(time.Since(kr.start).Nanoseconds()-(pulse*kr.pulseTime.Nanoseconds())) / float64(kr.pulseTime.Nanoseconds())
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
		pa.SetOne(i, pixarray.Pixel{v, 0, 0})
	}
	return time.Millisecond
}
