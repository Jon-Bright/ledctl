package pixarray

type baseArray struct {
	numPixels int
	pixels    []byte
	g         int
	r         int
	b         int
}

func newBaseArray(numPixels int, pixels []byte, order int) *baseArray {
	offsets := offsets[order]
	ba := baseArray{numPixels, pixels, offsets[0], offsets[1], offsets[2]}
	return &ba
}

func (ba *baseArray) NumPixels() int {
	return ba.numPixels
}

func (ba *baseArray) GetPixels() []Pixel {
	p := make([]Pixel, ba.numPixels)
	for i, v := range ba.pixels {
		switch i % 3 {
		case ba.g:
			p[i/3].G = int(v) & 0x7f
		case ba.r:
			p[i/3].R = int(v) & 0x7f
		case ba.b:
			p[i/3].B = int(v) & 0x7f
		}
	}
	return p
}

func (ba *baseArray) GetPixel(i int) Pixel {
	return Pixel{int(ba.pixels[i*3+ba.r]) & 0x7f, int(ba.pixels[i*3+ba.g]) & 0x7f, int(ba.pixels[i*3+ba.b]) & 0x7f}
}

func (ba *baseArray) SetAlternate(num int, div int, p1 Pixel, p2 Pixel) {
	totSet := 0
	shouldSet := 0
	for i := 0; i < ba.numPixels; i++ {
		shouldSet += num
		e1 := abs((totSet + div) - shouldSet)
		e2 := abs(totSet - shouldSet)
		if e1 < e2 {
			totSet += div
			ba.pixels[i*3+ba.g] = byte(0x80 | p2.G)
			ba.pixels[i*3+ba.r] = byte(0x80 | p2.R)
			ba.pixels[i*3+ba.b] = byte(0x80 | p2.B)
		} else {
			ba.pixels[i*3+ba.g] = byte(0x80 | p1.G)
			ba.pixels[i*3+ba.r] = byte(0x80 | p1.R)
			ba.pixels[i*3+ba.b] = byte(0x80 | p1.B)
		}
	}
}

func (ba *baseArray) SetPerChanAlternate(num Pixel, div int, p1 Pixel, p2 Pixel) {
	totSet := Pixel{}
	shouldSet := Pixel{}
	for i := 0; i < ba.numPixels; i++ {
		shouldSet.R += num.R
		e1 := abs((totSet.R + div) - shouldSet.R)
		e2 := abs(totSet.R - shouldSet.R)
		if e1 < e2 {
			totSet.R += div
			ba.pixels[i*3+ba.r] = byte(0x80 | p2.R)
		} else {
			ba.pixels[i*3+ba.r] = byte(0x80 | p1.R)
		}
		shouldSet.G += num.G
		e1 = abs((totSet.G + div) - shouldSet.G)
		e2 = abs(totSet.G - shouldSet.G)
		if e1 < e2 {
			totSet.G += div
			ba.pixels[i*3+ba.g] = byte(0x80 | p2.G)
		} else {
			ba.pixels[i*3+ba.g] = byte(0x80 | p1.G)
		}
		shouldSet.B += num.B
		e1 = abs((totSet.B + div) - shouldSet.B)
		e2 = abs(totSet.B - shouldSet.B)
		if e1 < e2 {
			totSet.B += div
			ba.pixels[i*3+ba.b] = byte(0x80 | p2.B)
		} else {
			ba.pixels[i*3+ba.b] = byte(0x80 | p1.B)
		}
	}
}

func (ba *baseArray) SetAll(p Pixel) {
	for i := 0; i < ba.numPixels; i++ {
		ba.pixels[i*3+ba.g] = byte(0x80 | p.G)
		ba.pixels[i*3+ba.r] = byte(0x80 | p.R)
		ba.pixels[i*3+ba.b] = byte(0x80 | p.B)
	}
}

func (ba *baseArray) SetOne(i int, p Pixel) {
	ba.pixels[i*3+ba.g] = byte(0x80 | p.G)
	ba.pixels[i*3+ba.r] = byte(0x80 | p.R)
	ba.pixels[i*3+ba.b] = byte(0x80 | p.B)
}
