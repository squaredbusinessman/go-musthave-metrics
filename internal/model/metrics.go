package models

// MetricType - тип метрики.
type MetricType string

const (
	// MetricTypeGauge - тип метрики с плавающей точкой.
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeCounter - тип метрики-счетчика.
	MetricTypeCounter MetricType = "counter"
)

// Gauge - значение gauge-метрики.
type Gauge struct {
	Value float64
}

// NewGaugeReplace - заменяет текущее значение gauge-метрики.
func (g *Gauge) NewGaugeReplace(val float64) {
	g.Value = val
}

// Counter - значение counter-метрики.
type Counter struct {
	Value int64
}

// NewValueIncrement - увеличивает значение счетчика на переданное число.
func (c *Counter) NewValueIncrement(val int64) {
	c.Value += val
}

// Metrics - JSON-представление метрики для HTTP API.
// Delta и Value сделаны указателями, чтобы отличать ноль от отсутствующего поля.
type Metrics struct {
	ID    string     `json:"id"`
	MType MetricType `json:"type"`
	Delta *int64     `json:"delta,omitempty"`
	Value *float64   `json:"value,omitempty"`
	Hash  string     `json:"hash,omitempty"`
}

// Metric - плоская модель метрики для строковых HTTP-эндпоинтов.
type Metric struct {
	Type  MetricType
	Name  string
	Value string
}
