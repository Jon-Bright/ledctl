package pixarray

import (
	"testing"
)

type testLeds struct {
	pixels []Pixel
}

func (l *testLeds) GetPixel(i int) Pixel {
	return pixels[i]
}

func (l *testLeds) SetPixel(i int, p Pixel) {
	pixels[i] = p
}

func newTestLeds(numPixels int) {
	return &testLeds{make([]Pixel, numPixels)}
}

func TestSetOneThenGetOneByOne(t *testing.T) {
	pa := NewPixArray(100, 3, newTestLeds(100))
	ps := Pixel{10, 25, 45}
	pb := Pixel{0, 0, 0}
	pa.SetOne(20, ps)
	for i := 0; i < 100; i++ {
		pg := pa.GetPixel(i)
		if i == 20 && pg != ps {
			t.Errorf("Set pixel incorrect, got: %v, want %v", pg, ps)
		} else if i != 20 && pg != pb {
			t.Errorf("Unset pixel incorrect, got: %v, want %v", pg, pb)
		}
	}
}

func TestSetOneThenGetAll(t *testing.T) {
	pa := NewPixArray(100, 3, newTestLeds(100))
	ps := Pixel{10, 25, 45}
	pb := Pixel{0, 0, 0}
	pa.SetOne(20, ps)
	py := pa.GetPixels()
	if len(py) != 100 {
		t.Errorf("Incorrect array len, got: %d, want: 100", len(py))
	}
	for i := 0; i < 100; i++ {
		if i == 20 && py[i] != ps {
			t.Errorf("Set pixel incorrect, got: %v, want %v", py[i], ps)
		} else if i != 20 && py[i] != pb {
			t.Errorf("Unset pixel incorrect, got: %v, want %v", py[i], pb)
		}
	}
}

func TestSetAlternate(t *testing.T) {
	pa := NewPixArray(100, 3, newTestLeds(100))
	p1 := Pixel{10, 25, 45}
	p2 := Pixel{9, 7, 5}

	tests := []struct {
		num   int
		div   int
		want1 int // Total number of p1 we expect over 100 pixels
		want2 int // Total number of p2 we expect over 100 pixels
		cons1 int // Max number of consecutive p1 we expect
		cons2 int // Max number of consecutive p2 we expect
	}{
		{9, 10, 10, 90, 1, 9},
		{5, 10, 50, 50, 1, 1},
		{51, 100, 49, 51, 1, 2},
		{52, 100, 48, 52, 1, 2},
		{5, 7, 29, 71, 1, 3},
	}

	for _, test := range tests {
		pa.SetAlternate(test.num, test.div, p1, p2)
		py := pa.GetPixels()
		if len(py) != 100 {
			t.Errorf("(%d/%d): Incorrect array len, got: %d, want: 100", test.num, test.div, len(py))
		}
		lp := Pixel{}
		n1 := 0
		n2 := 0
		cons := 0
		cons1 := 0
		cons2 := 0
		for i := 0; i < 100; i++ {
			if py[i] == lp {
				cons++
			} else {
				cons = 1
			}
			if py[i] == p1 {
				n1++
				if cons > cons1 {
					cons1 = cons
				}
			} else if py[i] == p2 {
				n2++
				if cons > cons2 {
					cons2 = cons
				}
			} else {
				t.Errorf("(%d/%d): Unexpected pixel got: %v, want: %v or %v", test.num, test.div, py[i], p1, p2)
			}
			lp = py[i]
		}
		if n1 != test.want1 {
			t.Errorf("(%d/%d): Wrong pixel1 count, got: %v, want %v", test.num, test.div, n1, test.want1)
		}
		if n2 != test.want2 {
			t.Errorf("(%d/%d): Wrong pixel2 count, got: %v, want %v", test.num, test.div, n2, test.want2)
		}
		if cons1 != test.cons1 {
			t.Errorf("(%d/%d): Wrong pixel1 consecutive count, got: %v, want %v", test.num, test.div, cons1, test.cons1)
		}
		if cons2 != test.cons2 {
			t.Errorf("(%d/%d): Wrong pixel2 consecutive count, got: %v, want %v", test.num, test.div, cons2, test.cons2)
		}
	}
}

func TestSetPerChanAlternate(t *testing.T) {
	pa := NewPixArray(100, 3, newTestLeds(100))
	p1 := Pixel{10, 25, 45}
	p2 := Pixel{9, 7, 5}

	tests := []struct {
		num   Pixel
		div   int
		want1 Pixel // Total number of p1 we expect over 100 pixels
		want2 Pixel // Total number of p2 we expect over 100 pixels
		cons1 Pixel // Max number of consecutive p1 we expect
		cons2 Pixel // Max number of consecutive p2 we expect
	}{
		{Pixel{9, 5, 1}, 10, Pixel{10, 50, 90}, Pixel{90, 50, 10}, Pixel{1, 1, 9}, Pixel{9, 1, 1}},
		{Pixel{51, 52, 99}, 100, Pixel{49, 48, 1}, Pixel{51, 52, 99}, Pixel{1, 1, 1}, Pixel{2, 2, 50}},
	}

	for _, test := range tests {
		pa.SetPerChanAlternate(test.num, test.div, p1, p2)
		py := pa.GetPixels()
		if len(py) != 100 {
			t.Errorf("(%d/%d): Incorrect array len, got: %d, want: 100", test.num, test.div, len(py))
		}
		lp := Pixel{}
		n1 := Pixel{}
		n2 := Pixel{}
		cons := Pixel{}
		cons1 := Pixel{}
		cons2 := Pixel{}
		for i := 0; i < 100; i++ {
			if py[i].R == lp.R {
				cons.R++
			} else {
				cons.R = 1
			}
			if py[i].G == lp.G {
				cons.G++
			} else {
				cons.G = 1
			}
			if py[i].B == lp.B {
				cons.B++
			} else {
				cons.B = 1
			}
			if py[i].R == p1.R {
				n1.R++
				if cons.R > cons1.R {
					cons1.R = cons.R
				}
			} else if py[i].R == p2.R {
				n2.R++
				if cons.R > cons2.R {
					cons2.R = cons.R
				}
			} else {
				t.Errorf("R(%d/%d): Unexpected pixel got: %v, want: %v or %v", test.num.R, test.div, py[i].R, p1.R, p2.R)
			}
			if py[i].G == p1.G {
				n1.G++
				if cons.G > cons1.G {
					cons1.G = cons.G
				}
			} else if py[i].G == p2.G {
				n2.G++
				if cons.G > cons2.G {
					cons2.G = cons.G
				}
			} else {
				t.Errorf("G(%d/%d): Unexpected pixel got: %v, want: %v or %v", test.num.G, test.div, py[i].G, p1.G, p2.G)
			}
			if py[i].B == p1.B {
				n1.B++
				if cons.B > cons1.B {
					cons1.B = cons.B
				}
			} else if py[i].B == p2.B {
				n2.B++
				if cons.B > cons2.B {
					cons2.B = cons.B
				}
			} else {
				t.Errorf("B(%d/%d): Unexpected pixel got: %v, want: %v or %v", test.num.B, test.div, py[i].B, p1.B, p2.B)
			}
			lp = py[i]
		}
		if n1 != test.want1 {
			t.Errorf("(%d/%d): Wrong pixel1 count, got: %v, want %v", test.num, test.div, n1, test.want1)
		}
		if n2 != test.want2 {
			t.Errorf("(%d/%d): Wrong pixel2 count, got: %v, want %v", test.num, test.div, n2, test.want2)
		}
		if cons1 != test.cons1 {
			t.Errorf("(%d/%d): Wrong pixel1 consecutive count, got: %v, want %v", test.num, test.div, cons1, test.cons1)
		}
		if cons2 != test.cons2 {
			t.Errorf("(%d/%d): Wrong pixel2 consecutive count, got: %v, want %v", test.num, test.div, cons2, test.cons2)
		}
	}
}

func BenchmarkSetAlternate(b *testing.B) {
	pa := NewPixArray(100, 3, newTestLeds(100))
	p1 := Pixel{10, 25, 45}
	p2 := Pixel{9, 7, 5}
	for i := 0; i < b.N/2; i++ {
		pa.SetAlternate(5, 7, p1, p2)
		pa.SetAlternate(2, 7, p1, p2)
	}
}

func BenchmarkSetPerChanAlternate(b *testing.B) {
	pa := NewPixArray(100, 3, newTestLeds(100))
	p1 := Pixel{10, 25, 45}
	p2 := Pixel{9, 7, 5}
	s1 := Pixel{5, 1, 2}
	s2 := Pixel{2, 6, 5}
	for i := 0; i < b.N/2; i++ {
		pa.SetPerChanAlternate(s1, 7, p1, p2)
		pa.SetPerChanAlternate(s2, 7, p1, p2)
	}
}
