package models

import "testing"

func TestGauge_NewGaugeReplace(t *testing.T) {
	g := &Gauge{Value: 1.0}
	g.NewGaugeReplace(2.5)

	if g.Value != 2.5 {
		t.Fatalf("got %v, want 2.5", g.Value)
	}
}

func TestCounter_NewValueIncrement(t *testing.T) {
	c := &Counter{Value: 0}
	c.NewValueIncrement(3)
	c.NewValueIncrement(2)

	if c.Value != 5 {
		t.Fatalf("got %v, want 5", c.Value)
	}
}
