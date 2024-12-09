package memstorage

import "metrics"

type Storage interface {
	UpdateGauge(name string, value float64)
}

type MemStorage struct {
	Gauges   map[string]metrics.Gauge
	Counters map[string]metrics.Counter
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	if g, ok := ms.Gauges[name]; ok {
		g.Update(value)
		return
	}
	g := metrics.Gauge{Name: name, Value: value}
	ms.Gauges[name] = g
}

func (ms *MemStorage) UpdateCounter(name string, value int64) {
	if c, ok := ms.Counters[name]; ok {
		c.Update(value)
		return
	}
	c := metrics.Counter{Name: name, Value: value}
	ms.Counters[name] = c
}
