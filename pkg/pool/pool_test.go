package pool

import "testing"

type resettableSample struct {
	buf   []byte
	count int
}

func (s *resettableSample) Reset() {
	s.buf = s.buf[:0]
	s.count = 0
}

func TestNewReturnsPool(t *testing.T) {
	if New[*resettableSample]() == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetReturnsZeroValueForEmptyPool(t *testing.T) {
	p := New[*resettableSample]()

	if got := p.Get(); got != nil {
		t.Fatalf("Get() from empty pool = %v, want nil", got)
	}
}

func TestPutResetsObjectBeforeStoring(t *testing.T) {
	p := New[*resettableSample]()
	item := &resettableSample{
		buf:   []byte{1, 2, 3},
		count: 42,
	}

	p.Put(item)

	got := p.Get()
	if got != item {
		t.Fatalf("Get() returned %p, want %p", got, item)
	}
	if got.count != 0 {
		t.Fatalf("count after reset = %d, want 0", got.count)
	}
	if len(got.buf) != 0 {
		t.Fatalf("buffer length after reset = %d, want 0", len(got.buf))
	}
}

func TestGetUsesLIFOOrder(t *testing.T) {
	p := New[*resettableSample]()
	first := &resettableSample{count: 1}
	second := &resettableSample{count: 2}

	p.Put(first)
	p.Put(second)

	if got := p.Get(); got != second {
		t.Fatalf("first Get() = %p, want %p", got, second)
	}
	if got := p.Get(); got != first {
		t.Fatalf("second Get() = %p, want %p", got, first)
	}
}
