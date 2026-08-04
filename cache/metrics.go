package cache

import (
	pmetrics "github.com/awslabs/operatorpkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	MetricSubsystem = "cache"

	// MetricLabelName identifies the logical cache instance, e.g. "aws.ssm" or
	// "nodeclaim.launch". This is set explicitly at construction (see New) and
	// MUST be low-cardinality: never a cache key, hash, or other unbounded value.
	MetricLabelName = "name"
	// MetricLabelResult carries the outcome of an operation: "hit"/"miss" for
	// gets and "added"/"exists" for adds.
	MetricLabelResult = "result"

	ResultHit    = "hit"
	ResultMiss   = "miss"
	ResultAdded  = "added"
	ResultExists = "exists"
)

// FlushSizeBuckets covers cache sizes from a handful of entries (e.g. ICE
// offerings) up to the tens-of-thousands (e.g. instance types): 1, 4, 16, 64,
// 256, 1k, 4k, 16k. Tune if a specific cache needs finer resolution.
var FlushSizeBuckets = prometheus.ExponentialBuckets(1, 4, 8)

// These are package-level so every instrumented cache shares one metric family,
// distinguished by the "name" label. Cardinality is bounded by the number of
// distinct cache instances (~25 across karpenter today), times the small
// result/type enums, so the total series count stays well under a thousand.
var (
	getsTotal = pmetrics.NewPrometheusCounter(
		metrics.Registry,
		prometheus.CounterOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: MetricSubsystem,
			Name:      "gets_total",
			Help:      "The number of cache Get calls, partitioned by result (hit/miss).",
		},
		[]string{MetricLabelName, MetricLabelResult},
	)

	addsTotal = pmetrics.NewPrometheusCounter(
		metrics.Registry,
		prometheus.CounterOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: MetricSubsystem,
			Name:      "adds_total",
			Help:      "The number of cache Add calls, partitioned by result. result=exists indicates a suppressed duplicate (dedupe hit).",
		},
		[]string{MetricLabelName, MetricLabelResult},
	)

	evictionsTotal = pmetrics.NewPrometheusCounter(
		metrics.Registry,
		prometheus.CounterOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: MetricSubsystem,
			Name:      "evictions_total",
			Help:      "The number of per-key removals, whether by TTL expiry or explicit Delete. Whole-cache flushes are tracked separately by flushes_total.",
		},
		[]string{MetricLabelName},
	)

	flushesTotal = pmetrics.NewPrometheusCounter(
		metrics.Registry,
		prometheus.CounterOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: MetricSubsystem,
			Name:      "flushes_total",
			Help:      "The number of times the whole cache was flushed.",
		},
		[]string{MetricLabelName},
	)

	flushSize = pmetrics.NewPrometheusHistogram(
		metrics.Registry,
		prometheus.HistogramOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: MetricSubsystem,
			Name:      "flush_size",
			Help:      "The number of entries discarded per whole-cache Flush.",
			Buckets:   FlushSizeBuckets,
		},
		[]string{MetricLabelName},
	)

	entries = pmetrics.NewPrometheusGauge(
		metrics.Registry,
		prometheus.GaugeOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: MetricSubsystem,
			Name:      "entries",
			Help:      "The current number of entries in the cache.",
		},
		[]string{MetricLabelName},
	)
)
