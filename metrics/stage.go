package metrics

// Stage describes the API stability of a metric, mirroring the Kubernetes
// metric stability levels. Declaring it lets a metrics documentation generator
// surface which metrics are safe to depend on and which may still change.
type Stage string

const (
	// Alpha marks an experimental metric that may change or be removed without
	// notice.
	Alpha Stage = "alpha"
	// Beta marks a metric that is fairly stable but whose name, labels, or
	// semantics may still change before it is promoted.
	Beta Stage = "beta"
	// GA marks a stable metric that is safe to depend on; breaking changes
	// follow the usual deprecation process.
	GA Stage = "ga"
)
