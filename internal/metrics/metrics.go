package metrics

type Gauge struct {
	Name  string
	Value float64
}

type Counter struct {
	Name  string
	Value int64
}

func (g *Gauge) Update(value float64) {
	g.Value = value
}

func (c *Counter) Update(value int64) {
	c.Value += value
}
