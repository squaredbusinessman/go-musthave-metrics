package audit

import (
	"context"
	"errors"
	"sync"
)

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Notify(ctx context.Context, event Event) error
}

type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
}

func NewPublisher(observers ...Observer) *Publisher {
	p := &Publisher{}
	for _, observer := range observers {
		p.Subscribe(observer)
	}
	return p
}

func (p *Publisher) Subscribe(observer Observer) {
	if observer == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

func (p *Publisher) Notify(ctx context.Context, event Event) error {
	p.mu.RLock()
	// Работаем с копией списка, чтобы рассылка не зависела от возможных новых подписок.
	observers := append([]Observer(nil), p.observers...)
	p.mu.RUnlock()

	var notifyErr error
	for _, observer := range observers {
		if err := observer.Notify(ctx, event); err != nil {
			notifyErr = errors.Join(notifyErr, err)
		}
	}

	return notifyErr
}

func (p *Publisher) HasObservers() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.observers) > 0
}
