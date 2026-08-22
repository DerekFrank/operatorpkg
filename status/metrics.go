package status

import (
	"fmt"

	pmetrics "github.com/awslabs/operatorpkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	MetricLabelNamespace       = "namespace"
	MetricLabelName            = "name"
	MetricLabelConditionStatus = "status"
	// MetricLabelToConditionStatus labels the transition_seconds histogram with
	// the status the condition transitioned TO. Combined with the existing
	// "status" label (the status being left), this lets consumers distinguish
	// how a dwell ended -- e.g. how long a condition was unhealthy
	// (status="False") before it recovered (to_status="True").
	MetricLabelToConditionStatus = "to_status"
)

const (
	MetricSubsystem      = "status_condition"
	TerminationSubsystem = "termination"
)

// Package-local metric dimensions specific to status-condition metrics. These
// are co-located with their name consts (rather than in the shared metrics
// package) because they are only emitted by the status controllers. The
// label-name consts above remain the values used in the metric label-names
// slices; these Label vars carry the documentation keyed by the same name.
var (
	Namespace = pmetrics.Label{
		Name: MetricLabelNamespace,
		Help: "The namespace of the object the metric describes.",
	}
	Name = pmetrics.Label{
		Name: MetricLabelName,
		Help: "The name of the object the metric describes.",
	}
	ConditionStatus = pmetrics.Label{
		Name: MetricLabelConditionStatus,
		Help: "The status of the condition (the state being left, for transition metrics).",
		Values: []string{
			string(metav1.ConditionTrue),
			string(metav1.ConditionFalse),
			string(metav1.ConditionUnknown),
		},
	}
	ToConditionStatus = pmetrics.Label{
		Name: MetricLabelToConditionStatus,
		Help: "The status the condition transitioned to, for transition metrics.",
		Values: []string{
			string(metav1.ConditionTrue),
			string(metav1.ConditionFalse),
			string(metav1.ConditionUnknown),
		},
	}
)

// Cardinality is limited to # objects * # conditions * # objectives
var ConditionDuration = conditionDurationMetric("", nil, pmetrics.LabelGroup, pmetrics.LabelKind)

func conditionDurationMetric(objectName string, buckets []float64, additionalLabels ...string) pmetrics.ObservationMetric {
	subsystem := lo.Ternary(len(objectName) == 0, MetricSubsystem, fmt.Sprintf("%s_%s", objectName, MetricSubsystem))
	buckets = lo.Ternary(len(buckets) == 0, prometheus.DefBuckets, buckets)

	return pmetrics.NewPrometheusHistogram(
		metrics.Registry,
		prometheus.HistogramOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: subsystem,
			Name:      "transition_seconds",
			Help:      "The amount of time a condition was in a given state (status) before transitioning to another state (to_status). e.g. Alarm := P99(Updated=False) > 5 minutes",
			Buckets:   buckets,
		},
		append([]string{
			pmetrics.LabelType,
			MetricLabelConditionStatus,
			MetricLabelToConditionStatus,
		}, additionalLabels...),
	)
}

// Cardinality is limited to # objects * # conditions
var ConditionCount = conditionCountMetric("", pmetrics.LabelGroup, pmetrics.LabelKind)

func conditionCountMetric(objectName string, additionalLabels ...string) pmetrics.GaugeMetric {
	subsystem := lo.Ternary(len(objectName) == 0, MetricSubsystem, fmt.Sprintf("%s_%s", objectName, MetricSubsystem))

	return pmetrics.NewPrometheusGauge(
		metrics.Registry,
		prometheus.GaugeOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: subsystem,
			Name:      "count",
			Help:      "The number of a condition for a given object, type and status. e.g. Alarm := Available=False > 0",
		},
		append([]string{
			MetricLabelNamespace,
			MetricLabelName,
			pmetrics.LabelType,
			MetricLabelConditionStatus,
			pmetrics.LabelReason,
		}, additionalLabels...),
	)
}

// Cardinality is limited to # objects * # conditions
// NOTE: This metric is based on a requeue so it won't show the current status seconds with extremely high accuracy.
// This metric is useful for aggregations. If you need a high accuracy metric, use operator_status_condition_last_transition_time_seconds
var ConditionCurrentStatusSeconds = conditionCurrentStatusSecondsMetric("", pmetrics.LabelGroup, pmetrics.LabelKind)

func conditionCurrentStatusSecondsMetric(objectName string, additionalLabels ...string) pmetrics.GaugeMetric {
	subsystem := lo.Ternary(len(objectName) == 0, MetricSubsystem, fmt.Sprintf("%s_%s", objectName, MetricSubsystem))

	return pmetrics.NewPrometheusGauge(
		metrics.Registry,
		prometheus.GaugeOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: subsystem,
			Name:      "current_status_seconds",
			Help:      "The current amount of time in seconds that a status condition has been in a specific state. Alarm := P99(Updated=Unknown) > 5 minutes",
		},
		append([]string{
			MetricLabelNamespace,
			MetricLabelName,
			pmetrics.LabelType,
			MetricLabelConditionStatus,
			pmetrics.LabelReason,
		}, additionalLabels...),
	)
}

// Cardinality is limited to # objects * # conditions
var ConditionTransitionsTotal = conditionTransitionsTotalMetric("", pmetrics.LabelGroup, pmetrics.LabelKind)

func conditionTransitionsTotalMetric(objectName string, additionalLabels ...string) pmetrics.CounterMetric {
	subsystem := lo.Ternary(len(objectName) == 0, MetricSubsystem, fmt.Sprintf("%s_%s", objectName, MetricSubsystem))

	return pmetrics.NewPrometheusCounter(
		metrics.Registry,
		prometheus.CounterOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: subsystem,
			Name:      "transitions_total",
			Help:      "The count of transitions of a given object, type and status.",
		},
		append([]string{
			pmetrics.LabelType,
			MetricLabelConditionStatus,
			pmetrics.LabelReason,
		}, additionalLabels...),
	)

}

var TerminationCurrentTimeSeconds = terminationCurrentTimeSecondsMetric("", pmetrics.LabelGroup, pmetrics.LabelKind)

func terminationCurrentTimeSecondsMetric(objectName string, additionalLabels ...string) pmetrics.GaugeMetric {
	subsystem := lo.Ternary(len(objectName) == 0, TerminationSubsystem, fmt.Sprintf("%s_%s", objectName, TerminationSubsystem))

	return pmetrics.NewPrometheusGauge(
		metrics.Registry,
		prometheus.GaugeOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: subsystem,
			Name:      "current_time_seconds",
			Help:      "The current amount of time in seconds that an object has been in terminating state.",
		},
		append([]string{
			MetricLabelNamespace,
			MetricLabelName,
		}, additionalLabels...),
	)
}

var TerminationDuration = terminationDurationMetric("", nil, pmetrics.LabelGroup, pmetrics.LabelKind)

func terminationDurationMetric(objectName string, buckets []float64, additionalLabels ...string) pmetrics.ObservationMetric {
	subsystem := lo.Ternary(len(objectName) == 0, TerminationSubsystem, fmt.Sprintf("%s_%s", objectName, TerminationSubsystem))
	buckets = lo.Ternary(len(buckets) == 0, prometheus.DefBuckets, buckets)

	return pmetrics.NewPrometheusHistogram(
		metrics.Registry,
		prometheus.HistogramOpts{
			Namespace: pmetrics.Namespace,
			Subsystem: subsystem,
			Name:      "duration_seconds",
			Help:      "The amount of time taken by an object to terminate completely.",
			Buckets:   buckets,
		},
		additionalLabels,
	)
}
