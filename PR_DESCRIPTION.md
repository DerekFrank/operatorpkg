Migrates the status controller from the deprecated `record.EventRecorder`
(k8s.io/client-go/tools/record) to `events.EventRecorder`
(k8s.io/client-go/tools/events).

The old events API is deprecated in controller-runtime v0.23.0:
- controller-runtime commit adding the new events API and deprecating the old one: https://github.com/kubernetes-sigs/controller-runtime/commit/572fad4
- KEP-383 (New Event API GA): https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/383-new-event-api-ga

This unblocks downstream consumers (e.g. karpenter) from passing
`mgr.GetEventRecorder()` instead of the deprecated `mgr.GetEventRecorderFor()`,
which triggers SA1019 staticcheck failures.
