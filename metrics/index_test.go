package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestIndexForgetsDeletedSeries checks that every way of deleting a series also drops
// it from the index, which otherwise keeps a copy of its labels forever.
func TestIndexForgetsDeletedSeries(t *testing.T) {
	labels := map[string]string{"key": "a", "other": "x"}
	for name, del := range map[string]func(GaugeMetric){
		"Delete":             func(m GaugeMetric) { m.Delete(labels) },
		"DeleteByKey":        func(m GaugeMetric) { m.DeleteByKey(map[string]string{"key": "a"}) },
		"DeletePartialMatch": func(m GaugeMetric) { m.DeletePartialMatch(map[string]string{"other": "x"}) },
		"Reset":              func(m GaugeMetric) { m.Reset() },
	} {
		t.Run(name, func(t *testing.T) {
			g := NewPrometheusGauge(prometheus.NewRegistry(), prometheus.GaugeOpts{Name: "test", Help: "test"}, []Label{{Name: "key", Key: true}, {Name: "other"}}, Alpha)
			g.Set(1, labels)
			del(g)
			if n := len(g.(*PrometheusGauge).index.series); n != 0 {
				t.Fatalf("index has %d keys after %s, want 0", n, name)
			}
		})
	}
}
