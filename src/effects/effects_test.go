package effects

import (
	"math"
	"pixarray"
	"testing"
	"time"
)

type FakeLEDDev struct {
}

func (f *FakeLEDDev) Fd() uintptr {
	return 0
}

func (f *FakeLEDDev) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func d(s string, tb testing.TB) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		tb.Fatalf("Couldn't parse duration %s: %v", s, err)
	}
	return d
}

func TestAllSameFade(t *testing.T) {
	pa, err := pixarray.NewPixArray(&FakeLEDDev{}, 100, 0, pixarray.GRB)
	if err != nil {
		t.Fatalf("Failed NewPixArray: %v", err)
	}

	tests := []struct {
		start   pixarray.Pixel
		dest    pixarray.Pixel
		fadeLen time.Duration
		len     time.Duration
		r       float64
		g       float64
		b       float64
	}{
		{pixarray.Pixel{0, 0, 0}, pixarray.Pixel{127, 0, 0}, d("1.0s", t), d("0.5s", t), 63.5, 0, 0},
		{pixarray.Pixel{0, 127, 0}, pixarray.Pixel{127, 0, 0}, d("1.0s", t), d("0.5s", t), 63.5, 63.5, 0},
		{pixarray.Pixel{127, 127, 127}, pixarray.Pixel{127, 0, 127}, d("3.0s", t), d("1.0s", t), 127, 84.66666, 127},
		{pixarray.Pixel{127, 127, 127}, pixarray.Pixel{127, 0, 127}, d("3.0s", t), d("2.0s", t), 127, 42.33333, 127},
		{pixarray.Pixel{127, 127, 127}, pixarray.Pixel{0, 0, 0}, d("127.0s", t), d("10.5s", t), 116.5, 116.5, 116.5},
		{pixarray.Pixel{127, 127, 0}, pixarray.Pixel{0, 0, 127}, d("127.0s", t), d("10.5s", t), 116.5, 116.5, 10.5},
		{pixarray.Pixel{126, 126, 0}, pixarray.Pixel{0, 63, 126}, d("126.0s", t), d("10.5s", t), 115.5, 120.75, 10.5},
		{pixarray.Pixel{0, 0, 0}, pixarray.Pixel{120, 10, 0}, d("120.0s", t), d("6.0s", t), 6.0, 0.5, 0},
	}

	tm := time.Now()
	for _, test := range tests {
		pa.SetAll(test.start)
		f := NewFade(test.fadeLen, test.dest)
		f.Start(pa, tm)
		tm = tm.Add(test.len)
		f.NextStep(pa, tm)
		py := pa.GetPixels()
		totR := 0
		totG := 0
		totB := 0
		rc := int(math.Ceil(test.r))
		rf := int(math.Floor(test.r))
		gc := int(math.Ceil(test.g))
		gf := int(math.Floor(test.g))
		bc := int(math.Ceil(test.b))
		bf := int(math.Floor(test.b))
		for i, p := range py {
			totR += p.R
			totG += p.G
			totB += p.B
			if p.R != rc && p.R != rf {
				t.Errorf("Wrong red at pixel %d, want %d/%d, got %d", i, rc, rf, p.R)
			}
			if p.G != gc && p.G != gf {
				t.Errorf("Wrong green at pixel %d, want %d/%d, got %d", i, gc, gf, p.G)
			}
			if p.B != bc && p.B != bf {
				t.Errorf("Wrong blue at pixel %d, want %d/%d, got %d", i, bc, bf, p.B)
			}
		}
		dR := float64(totR) / float64(len(py))
		if math.Abs(dR-test.r) > 0.01 {
			t.Errorf("Wrong average red, want %f, got %f", test.r, dR)
		}
		dG := float64(totG) / float64(len(py))
		if math.Abs(dG-test.g) > 0.01 {
			t.Errorf("Wrong average green, want %f, got %f", test.g, dG)
		}
		dB := float64(totB) / float64(len(py))
		if math.Abs(dB-test.b) > 0.01 {
			t.Errorf("Wrong average blue, want %f, got %f", test.b, dB)
		}
	}
}

func BenchmarkFadeStep(b *testing.B) {
	pa, err := pixarray.NewPixArray(&FakeLEDDev{}, 100, 0, pixarray.GRB)
	if err != nil {
		b.Fatalf("Failed NewPixArray: %v", err)
	}
	pa.SetAll(pixarray.Pixel{127, 0, 0})
	tm := time.Now()
	add := time.Duration((7200 * time.Second).Nanoseconds() / int64(b.N))
	if add == 0 {
		b.Fatalf("Zero delay")
	}
	f := NewFade(d("7200.0s", b), pixarray.Pixel{0, 127, 0})
	f.Start(pa, tm)
	for i := 0; i < b.N; i++ {
		tm = tm.Add(add)
		f.NextStep(pa, tm)
	}
}
