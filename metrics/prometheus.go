package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// labelNames returns the Prometheus label-name slice for a set of Labels.
func labelNames(labels []Label) []string {
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.Name
	}
	return names
}

type PrometheusCounter struct {
	*prometheus.CounterVec
	labels []Label
	stage  Stage
	index  *keyIndex
}

func NewPrometheusCounter(registry prometheus.Registerer, opts prometheus.CounterOpts, labels []Label, stage Stage) CounterMetric {
	c := prometheus.NewCounterVec(opts, labelNames(labels))
	registry.MustRegister(c)
	return &PrometheusCounter{CounterVec: c, labels: labels, stage: stage, index: newKeyIndex(labels)}
}

func (pc *PrometheusCounter) Labels() []Label { return pc.labels }

func (pc *PrometheusCounter) Stage() Stage { return pc.stage }

func (pc *PrometheusCounter) Inc(labels map[string]string) {
	pc.index.emit(labels, func() { pc.CounterVec.With(labels).Inc() })
}

func (pc *PrometheusCounter) Add(v float64, labels map[string]string) {
	pc.index.emit(labels, func() { pc.CounterVec.With(labels).Add(v) })
}

func (pc *PrometheusCounter) Delete(labels map[string]string) {
	pc.index.delete(labels, func() { pc.CounterVec.Delete(labels) })
}

func (pc *PrometheusCounter) DeleteByKey(key map[string]string) {
	pc.index.deleteByKey(key, pc.CounterVec.Delete)
}

func (pc *PrometheusCounter) DeletePartialMatch(labels map[string]string) {
	pc.index.deletePartialMatch(labels, func() { pc.CounterVec.DeletePartialMatch(labels) })
}

func (pc *PrometheusCounter) Reset() {
	pc.index.reset(pc.CounterVec.Reset)
}

type PrometheusGauge struct {
	*prometheus.GaugeVec
	labels []Label
	stage  Stage
	index  *keyIndex
}

func NewPrometheusGauge(registry prometheus.Registerer, opts prometheus.GaugeOpts, labels []Label, stage Stage) GaugeMetric {
	g := prometheus.NewGaugeVec(opts, labelNames(labels))
	registry.MustRegister(g)
	return &PrometheusGauge{GaugeVec: g, labels: labels, stage: stage, index: newKeyIndex(labels)}
}

func (pg *PrometheusGauge) Labels() []Label { return pg.labels }

func (pg *PrometheusGauge) Stage() Stage { return pg.stage }

func (pg *PrometheusGauge) Set(v float64, labels map[string]string) {
	pg.index.emit(labels, func() { pg.GaugeVec.With(labels).Set(v) })
}

func (pg *PrometheusGauge) Delete(labels map[string]string) {
	pg.index.delete(labels, func() { pg.GaugeVec.Delete(labels) })
}

func (pg *PrometheusGauge) DeleteByKey(key map[string]string) {
	pg.index.deleteByKey(key, pg.GaugeVec.Delete)
}

func (pg *PrometheusGauge) ReplaceByKey(key map[string]string, series ...Sample) {
	pg.index.replace(key, series, func(s Sample) { pg.GaugeVec.With(s.Labels).Set(s.Value) }, pg.GaugeVec.Delete)
}

func (pg *PrometheusGauge) DeletePartialMatch(labels map[string]string) {
	pg.index.deletePartialMatch(labels, func() { pg.GaugeVec.DeletePartialMatch(labels) })
}

func (pg *PrometheusGauge) Reset() {
	pg.index.reset(pg.GaugeVec.Reset)
}

type PrometheusHistogram struct {
	*prometheus.HistogramVec
	labels []Label
	stage  Stage
	index  *keyIndex
}

func NewPrometheusHistogram(registry prometheus.Registerer, opts prometheus.HistogramOpts, labels []Label, stage Stage) ObservationMetric {
	h := prometheus.NewHistogramVec(opts, labelNames(labels))
	registry.MustRegister(h)
	return &PrometheusHistogram{HistogramVec: h, labels: labels, stage: stage, index: newKeyIndex(labels)}
}

func (ph *PrometheusHistogram) Labels() []Label { return ph.labels }

func (ph *PrometheusHistogram) Stage() Stage { return ph.stage }

func (ph *PrometheusHistogram) Observe(v float64, labels map[string]string) {
	ph.index.emit(labels, func() { ph.HistogramVec.With(labels).Observe(v) })
}

func (ph *PrometheusHistogram) Delete(labels map[string]string) {
	ph.index.delete(labels, func() { ph.HistogramVec.Delete(labels) })
}

func (ph *PrometheusHistogram) DeleteByKey(key map[string]string) {
	ph.index.deleteByKey(key, ph.HistogramVec.Delete)
}

func (ph *PrometheusHistogram) DeletePartialMatch(labels map[string]string) {
	ph.index.deletePartialMatch(labels, func() { ph.HistogramVec.DeletePartialMatch(labels) })
}

func (ph *PrometheusHistogram) Reset() {
	ph.index.reset(ph.HistogramVec.Reset)
}

type PrometheusSummary struct {
	*prometheus.SummaryVec
	labels []Label
	stage  Stage
	index  *keyIndex
}

func NewPrometheusSummary(registry prometheus.Registerer, opts prometheus.SummaryOpts, labels []Label, stage Stage) ObservationMetric {
	s := prometheus.NewSummaryVec(opts, labelNames(labels))
	registry.MustRegister(s)
	return &PrometheusSummary{SummaryVec: s, labels: labels, stage: stage, index: newKeyIndex(labels)}
}

func (ps *PrometheusSummary) Labels() []Label { return ps.labels }

func (ps *PrometheusSummary) Stage() Stage { return ps.stage }

func (ps *PrometheusSummary) Observe(v float64, labels map[string]string) {
	ps.index.emit(labels, func() { ps.SummaryVec.With(labels).Observe(v) })
}

func (ps *PrometheusSummary) Delete(labels map[string]string) {
	ps.index.delete(labels, func() { ps.SummaryVec.Delete(labels) })
}

func (ps *PrometheusSummary) DeleteByKey(key map[string]string) {
	ps.index.deleteByKey(key, ps.SummaryVec.Delete)
}

func (ps *PrometheusSummary) DeletePartialMatch(labels map[string]string) {
	ps.index.deletePartialMatch(labels, func() { ps.SummaryVec.DeletePartialMatch(labels) })
}

func (ps *PrometheusSummary) Reset() {
	ps.index.reset(ps.SummaryVec.Reset)
}
