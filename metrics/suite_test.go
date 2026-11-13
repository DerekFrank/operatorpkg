package metrics_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/awslabs/operatorpkg/metrics"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics")
}

var (
	key      = metrics.Label{Name: "key", Key: true}
	otherKey = metrics.Label{Name: "other_key", Key: true}
	name     = metrics.Label{Name: "name"}
	other    = metrics.Label{Name: "other"}
)

type metric interface {
	Delete(labels map[string]string)
	DeleteByKey(key map[string]string)
	DeletePartialMatch(labels map[string]string)
	Reset()
}

type constructor func(r prometheus.Registerer, labels []metrics.Label) (metric, func(map[string]string), prometheus.Collector)

func gauge(r prometheus.Registerer, labels []metrics.Label) (metric, func(map[string]string), prometheus.Collector) {
	g := metrics.NewPrometheusGauge(r, prometheus.GaugeOpts{Name: "test_gauge", Help: "test"}, labels, metrics.Alpha)
	return g, func(l map[string]string) { g.Set(1, l) }, g.(*metrics.PrometheusGauge).GaugeVec
}

func counter(r prometheus.Registerer, labels []metrics.Label) (metric, func(map[string]string), prometheus.Collector) {
	c := metrics.NewPrometheusCounter(r, prometheus.CounterOpts{Name: "test_counter", Help: "test"}, labels, metrics.Alpha)
	return c, func(l map[string]string) { c.Inc(l) }, c.(*metrics.PrometheusCounter).CounterVec
}

func histogram(r prometheus.Registerer, labels []metrics.Label) (metric, func(map[string]string), prometheus.Collector) {
	h := metrics.NewPrometheusHistogram(r, prometheus.HistogramOpts{Name: "test_histogram", Help: "test"}, labels, metrics.Alpha)
	return h, func(l map[string]string) { h.Observe(1, l) }, h.(*metrics.PrometheusHistogram).HistogramVec
}

func summary(r prometheus.Registerer, labels []metrics.Label) (metric, func(map[string]string), prometheus.Collector) {
	s := metrics.NewPrometheusSummary(r, prometheus.SummaryOpts{Name: "test_summary", Help: "test"}, labels, metrics.Alpha)
	return s, func(l map[string]string) { s.Observe(1, l) }, s.(*metrics.PrometheusSummary).SummaryVec
}

var _ = Describe("DeleteByKey", func() {
	DescribeTable("should delete only the key's series",
		func(newMetric constructor, labels []metrics.Label, emitted []map[string]string, key map[string]string, remaining int) {
			m, emit, collector := newMetric(prometheus.NewRegistry(), labels)
			for _, l := range emitted {
				emit(l)
			}
			Expect(testutil.CollectAndCount(collector)).To(Equal(len(emitted)))
			m.DeleteByKey(key)
			Expect(testutil.CollectAndCount(collector)).To(Equal(remaining))
		},
		func(newMetric constructor, labels []metrics.Label, emitted []map[string]string, key map[string]string, remaining int) string {
			return fmt.Sprintf("%v deleting %v leaves %d", labels, key, remaining)
		},
		Entry(nil, gauge, []metrics.Label{key, other}, []map[string]string{
			{"key": "a", "other": "x"}, {"key": "a", "other": "y"}, {"key": "b", "other": "x"},
		}, map[string]string{"key": "a"}, 1),
		Entry(nil, counter, []metrics.Label{key, other}, []map[string]string{
			{"key": "a", "other": "x"}, {"key": "a", "other": "y"}, {"key": "b", "other": "x"},
		}, map[string]string{"key": "a"}, 1),
		Entry(nil, histogram, []metrics.Label{key, other}, []map[string]string{
			{"key": "a", "other": "x"}, {"key": "a", "other": "y"}, {"key": "b", "other": "x"},
		}, map[string]string{"key": "a"}, 1),
		Entry(nil, summary, []metrics.Label{key, other}, []map[string]string{
			{"key": "a", "other": "x"}, {"key": "a", "other": "y"}, {"key": "b", "other": "x"},
		}, map[string]string{"key": "a"}, 1),
		// a key of several dimensions
		Entry(nil, gauge, []metrics.Label{key, otherKey, other}, []map[string]string{
			{"key": "a", "other_key": "1", "other": "x"}, {"key": "a", "other_key": "1", "other": "y"},
			{"key": "a", "other_key": "2", "other": "x"}, {"key": "b", "other_key": "1", "other": "x"},
		}, map[string]string{"key": "a", "other_key": "1"}, 2),
		// key values are separated, so ("a", "b1") is not ("ab", "1")
		Entry(nil, gauge, []metrics.Label{key, otherKey, other}, []map[string]string{
			{"key": "a", "other_key": "b1", "other": "x"}, {"key": "ab", "other_key": "1", "other": "x"},
		}, map[string]string{"key": "ab", "other_key": "1"}, 1),
	)
	DescribeTable("should panic unless given exactly the Key labels",
		func(labels []metrics.Label, key map[string]string) {
			m, _, _ := gauge(prometheus.NewRegistry(), labels)
			Expect(func() { m.DeleteByKey(key) }).To(Panic())
		},
		Entry("no Key labels", []metrics.Label{name, other}, map[string]string{"name": "a"}),
		Entry("a Key label missing", []metrics.Label{key, otherKey}, map[string]string{"key": "a"}),
		Entry("a label that isn't Key", []metrics.Label{key, other}, map[string]string{"key": "a", "other": "x"}),
	)
	It("should not be affected by the caller changing the labels after emitting", func() {
		m, emit, collector := gauge(prometheus.NewRegistry(), []metrics.Label{key, other})
		labels := map[string]string{"key": "a", "other": "x"}
		emit(labels)
		labels["other"] = "y"
		m.DeleteByKey(map[string]string{"key": "a"})
		Expect(testutil.CollectAndCount(collector)).To(Equal(0))
	})
})

var _ = Describe("ReplaceByKey", func() {
	var g metrics.GaugeMetric
	var vec *prometheus.GaugeVec
	BeforeEach(func() {
		g = metrics.NewPrometheusGauge(prometheus.NewRegistry(), prometheus.GaugeOpts{Name: "test_gauge", Help: "test"}, []metrics.Label{key, other}, metrics.Alpha)
		vec = g.(*metrics.PrometheusGauge).GaugeVec
		g.Set(1, map[string]string{"key": "a", "other": "x"})
		g.Set(1, map[string]string{"key": "a", "other": "y"})
		g.Set(1, map[string]string{"key": "b", "other": "x"})
	})
	It("should set the series and delete the key's other series", func() {
		g.ReplaceByKey(map[string]string{"key": "a"}, metrics.Sample{Value: 2, Labels: map[string]string{"key": "a", "other": "y"}}, metrics.Sample{Value: 3, Labels: map[string]string{"key": "a", "other": "z"}})
		Expect(testutil.CollectAndCount(vec)).To(Equal(3))
		Expect(testutil.ToFloat64(vec.With(map[string]string{"key": "a", "other": "y"}))).To(Equal(2.0))
		Expect(testutil.ToFloat64(vec.With(map[string]string{"key": "a", "other": "z"}))).To(Equal(3.0))
		Expect(testutil.ToFloat64(vec.With(map[string]string{"key": "b", "other": "x"}))).To(Equal(1.0))
		// the index holds the replaced series, so DeleteByKey deletes them
		g.DeleteByKey(map[string]string{"key": "a"})
		Expect(testutil.CollectAndCount(vec)).To(Equal(1))
	})
	It("should keep the series when replaced with the same ones", func() {
		for range 2 {
			g.ReplaceByKey(map[string]string{"key": "a"}, metrics.Sample{Value: 2, Labels: map[string]string{"key": "a", "other": "x"}})
		}
		Expect(testutil.CollectAndCount(vec)).To(Equal(2))
		Expect(testutil.ToFloat64(vec.With(map[string]string{"key": "a", "other": "x"}))).To(Equal(2.0))
	})
	It("should delete every series of the key when given none", func() {
		g.ReplaceByKey(map[string]string{"key": "a"})
		Expect(testutil.CollectAndCount(vec)).To(Equal(1))
	})
	It("should panic when a series doesn't have the key's values", func() {
		Expect(func() {
			g.ReplaceByKey(map[string]string{"key": "a"}, metrics.Sample{Labels: map[string]string{"key": "b", "other": "x"}})
		}).To(Panic())
	})
})

func BenchmarkDelete(b *testing.B) {
	for _, series := range []int{1_000, 10_000} {
		for _, byKey := range []bool{false, true} {
			b.Run(fmt.Sprintf("series=%d/byKey=%t", series, byKey), func(b *testing.B) {
				g := metrics.NewPrometheusGauge(prometheus.NewRegistry(), prometheus.GaugeOpts{Name: "bench", Help: "bench"}, []metrics.Label{{Name: "name", Key: byKey}, other}, metrics.Alpha)
				for i := range series {
					g.Set(1, map[string]string{"name": fmt.Sprint(i), "other": "x"})
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					n := fmt.Sprint(i % series)
					if byKey {
						g.DeleteByKey(map[string]string{"name": n})
					} else {
						g.DeletePartialMatch(map[string]string{"name": n})
					}
					g.Set(1, map[string]string{"name": n, "other": "x"})
				}
			})
		}
	}
}

func BenchmarkSet(b *testing.B) {
	for _, l := range []metrics.Label{name, {Name: "name", Key: true}} {
		b.Run(fmt.Sprintf("key=%t", l.Key), func(b *testing.B) {
			g := metrics.NewPrometheusGauge(prometheus.NewRegistry(), prometheus.GaugeOpts{Name: "bench", Help: "bench"}, []metrics.Label{l, other}, metrics.Alpha)
			labels := map[string]string{"name": "a", "other": "x"}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				g.Set(1, labels)
			}
		})
	}
}
