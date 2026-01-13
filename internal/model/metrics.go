package models

const (
	MetricTypeGauge   = "gauge"
	MetricTypeCounter = "counter"
)

type Gauge struct {
	Value float64
}

func (g *Gauge) NewGaugeReplace(val float64) {
	g.Value = val
}

type Counter struct {
	Value int64
}

func (c *Counter) NewValueIncrement(val int64) {
	c.Value += val
}

// Metrics NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Ограничиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

type Metric struct {
	Type  string
	Name  string
	Value string
}
