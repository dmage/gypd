package mathgraph

import "testing"

func TestZero(t *testing.T) {
	var x Max
	if x.Value() != 0 {
		t.Errorf("x.Value() = %d; want 0", x.Value())
	}
}

func TestAddInt(t *testing.T) {
	var x Max
	x.Add(NewConst(1))
	if x.Value() != 1 {
		t.Errorf("x.Value() = %d; want 1", x.Value())
	}
	x.Add(NewConst(2))
	if x.Value() != 2 {
		t.Errorf("x.Value() = %d; want 2", x.Value())
	}
}

func TestLaziness(t *testing.T) {
	var x, y, z Max
	x.Add(&y)
	y.Add(&z)
	x.Add(NewConst(100))
	y.Add(NewConst(20))
	z.Add(NewConst(3))
	if x.Value() != 100 {
		t.Errorf("x.Value() = %d; want 100", x.Value())
	}
	y.Add(NewConst(200))
	if x.Value() != 200 {
		t.Errorf("x.Value() = %d; want 200", x.Value())
	}
}

func TestLoop(t *testing.T) {
	var x, y Max
	x.Add(&y)
	y.Add(&x)
	if x.Value() != 0 {
		t.Errorf("x.Value() = %d; want 0", x.Value())
	}
	x.Add(NewConst(10))
	y.Add(NewConst(2))
	if x.Value() != 10 {
		t.Errorf("x.Value() = %d; want 10", x.Value())
	}
	if y.Value() != 10 {
		t.Errorf("y.Value() = %d; want 10", y.Value())
	}
	y.Add(NewConst(11))
	if x.Value() != 11 {
		t.Errorf("x.Value() = %d; want 11", x.Value())
	}
	if y.Value() != 11 {
		t.Errorf("y.Value() = %d; want 11", y.Value())
	}
}
