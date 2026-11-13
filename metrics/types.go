package metrics

const (
	Namespace   = "operator"
	LabelGroup  = "group"
	LabelKind   = "kind"
	LabelType   = "type"
	LabelReason = "reason"
)

type ObservationMetric interface {
	Observe(v float64, labels map[string]string)
	Delete(labels map[string]string)
	// DeleteByKey deletes every series emitted with key, which gives exactly the
	// metric's Key labels. It panics if the metric has no Key labels.
	DeleteByKey(key map[string]string)
	// Deprecated: DeletePartialMatch scans every series of the metric. Use Delete
	// with the full label set, or mark the labels that identify a series' object as
	// Key and use DeleteByKey.
	DeletePartialMatch(labels map[string]string)
	Reset()
	// Labels returns the dimensions the metric was declared with.
	Labels() []Label
	// Stage returns the metric's API stability.
	Stage() Stage
}

type CounterMetric interface {
	Add(v float64, labels map[string]string)
	Inc(labels map[string]string)
	Delete(labels map[string]string)
	// DeleteByKey deletes every series emitted with key, which gives exactly the
	// metric's Key labels. It panics if the metric has no Key labels.
	DeleteByKey(key map[string]string)
	// Deprecated: DeletePartialMatch scans every series of the metric. Use Delete
	// with the full label set, or mark the labels that identify a series' object as
	// Key and use DeleteByKey.
	DeletePartialMatch(labels map[string]string)
	Reset()
	// Labels returns the dimensions the metric was declared with.
	Labels() []Label
	// Stage returns the metric's API stability.
	Stage() Stage
}

type GaugeMetric interface {
	Set(v float64, labels map[string]string)
	Delete(labels map[string]string)
	// DeleteByKey deletes every series emitted with key, which gives exactly the
	// metric's Key labels. It panics if the metric has no Key labels.
	DeleteByKey(key map[string]string)
	// ReplaceByKey sets series, which must all have the Key values in key, and then
	// deletes every other series emitted with key. It panics if the metric has no Key
	// labels.
	ReplaceByKey(key map[string]string, series ...Sample)
	// Deprecated: DeletePartialMatch scans every series of the metric. Use Delete
	// with the full label set, or mark the labels that identify a series' object as
	// Key and use DeleteByKey.
	DeletePartialMatch(labels map[string]string)
	Reset()
	// Labels returns the dimensions the metric was declared with.
	Labels() []Label
	// Stage returns the metric's API stability.
	Stage() Stage
}

// Sample is a gauge value and the labels to set it with.
type Sample struct {
	Value  float64
	Labels map[string]string
}
