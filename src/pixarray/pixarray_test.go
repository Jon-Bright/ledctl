package pixarray

import (
	"testing"
)

type FakeLEDDev struct {
}

func (f *FakeLEDDev) Fd() uintptr {
	return 0
}

func (f *FakeLEDDev) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func TestSetOneThenGetOneByOne(t *testing.T) {
	pa, err := NewPixArray(&FakeLEDDev{}, 100, 0, GRB)
	if err!=nil {
		t.Fatalf("Failed NewPixArray: %v", err)
	}
	ps:=Pixel{10,25,45}
	pb:=Pixel{0,0,0}
	pa.SetOne(20, ps)
	for i:=0; i<100; i++ {
		pg:=pa.GetPixel(i)
		if i==20 && pg!=ps {
			t.Errorf("Set pixel incorrect, got: %v, want %v", pg, ps)
		} else if i!=20 && pg!=pb {
			t.Errorf("Unset pixel incorrect, got: %v, want %v", pg, pb)
		}
	}
}

func TestSetOneThenGetAll(t *testing.T) {
	pa, err := NewPixArray(&FakeLEDDev{}, 100, 0, GRB)
	if err!=nil {
		t.Fatalf("Failed NewPixArray: %v", err)
	}
	ps:=Pixel{10,25,45}
	pb:=Pixel{0,0,0}
	pa.SetOne(20, ps)
	py:=pa.GetPixels()
	if len(py)!=100 {
		t.Errorf("Incorrect array len, got: %d, want: 100", len(py))
	}
	for i:=0; i<100; i++ {
		if i==20 && py[i]!=ps {
			t.Errorf("Set pixel incorrect, got: %v, want %v", py[i], ps)
		} else if i!=20 && py[i]!=pb {
			t.Errorf("Unset pixel incorrect, got: %v, want %v", py[i], pb)
		}
	}
}

func TestSetAlternate(t *testing.T) {
	pa, err := NewPixArray(&FakeLEDDev{}, 100, 0, GRB)
	if err!=nil {
		t.Fatalf("Failed NewPixArray: %v", err)
	}
	p1:=Pixel{10,25,45}
	p2:=Pixel{9,7,5}

	tests := []struct {
		num int
		div int
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

	for _,test:=range tests {
		pa.SetAlternate(test.num, test.div, p1, p2)
		py:=pa.GetPixels()
		if len(py)!=100 {
			t.Errorf("(%d/%d): Incorrect array len, got: %d, want: 100", test.num, test.div, len(py))
		}
		lp:=Pixel{}
		n1:=0
		n2:=0
		cons:=0
		cons1:=0
		cons2:=0
		for i:=0; i<100; i++ {
			if py[i]==lp {
				cons++
			} else {
				cons=1
			}
			if py[i]==p1 {
				n1++
				if cons>cons1 {
					cons1=cons
				}
			} else if py[i]==p2 {
				n2++
				if cons>cons2 {
					cons2=cons
				}
			} else {
				t.Errorf("(%d/%d): Unexpected pixel got: %v, want: %v or %v", test.num, test.div, py[i], p1, p2)
			}
			lp=py[i]
		}
		if n1!=test.want1 {
			t.Errorf("(%d/%d): Wrong pixel1 count, got: %v, want %v", test.num, test.div, n1, test.want1)
		}
		if n2!=test.want2 {
			t.Errorf("(%d/%d): Wrong pixel2 count, got: %v, want %v", test.num, test.div, n2, test.want2)
		}
		if cons1!=test.cons1 {
			t.Errorf("(%d/%d): Wrong pixel1 consecutive count, got: %v, want %v", test.num, test.div, cons1, test.cons1)
		}
		if cons2!=test.cons2 {
			t.Errorf("(%d/%d): Wrong pixel2 consecutive count, got: %v, want %v", test.num, test.div, cons2, test.cons2)
		}
	}
}

func BenchmarkSetAlternate(b *testing.B) {
	pa, err := NewPixArray(&FakeLEDDev{}, 100, 0, GRB)
	if err!=nil {
		b.Fatalf("Failed NewPixArray: %v", err)
	}
	p1:=Pixel{10,25,45}
	p2:=Pixel{9,7,5}
	for i := 0; i < b.N/2; i++ {
		pa.SetAlternate(5,7,p1,p2)
		pa.SetAlternate(2,7,p1,p2)
	}
}
