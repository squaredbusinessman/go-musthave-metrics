package pool

import "sync"

// Resetter ограничивает типы, которые умеют очищать своё состояние перед
// повторным использованием.
type Resetter interface {
	Reset()
}

// Pool хранит объекты одного типа и безопасен для конкурентного доступа.
type Pool[T Resetter] struct {
	mu    sync.Mutex
	items []T
}

// New создаёт пустой пул объектов.
func New[T Resetter]() *Pool[T] {
	return &Pool[T]{}
}

// Get возвращает объект из пула или zero value типа T, если пул пуст.
func (p *Pool[T]) Get() T {
	var zero T
	if p == nil {
		return zero
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	last := len(p.items) - 1
	if last < 0 {
		return zero
	}

	item := p.items[last]
	p.items[last] = zero
	p.items = p.items[:last]

	return item
}

// Put сбрасывает состояние объекта и помещает его обратно в пул.
func (p *Pool[T]) Put(item T) {
	if p == nil {
		return
	}

	item.Reset()

	p.mu.Lock()
	p.items = append(p.items, item)
	p.mu.Unlock()
}
