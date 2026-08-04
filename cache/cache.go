// Package cache wraps github.com/patrickmn/go-cache with Prometheus metrics.
//
// The wrapped Cache embeds *cache.Cache, so it is a near drop-in replacement:
// every method of the underlying cache is still available, and only the
// operations that carry an observable signal (Get, Set, SetDefault, Add,
// Delete, Flush) are overridden to record metrics. Construct one with New,
// passing a stable, low-cardinality name used as the "name" metric label.
package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// Item re-exports cache.Item so callers can migrate to this package without
// also importing go-cache directly (e.g. for the map returned by Items()).
type Item = cache.Item

// Re-export go-cache's sentinel expiration durations so callers don't need to
// import go-cache directly to construct a cache.
const (
	NoExpiration      = cache.NoExpiration
	DefaultExpiration = cache.DefaultExpiration
)

// Cache is an instrumented wrapper around *cache.Cache. The zero value is not
// usable; construct with New.
type Cache struct {
	*cache.Cache

	name string

	// userOnEvicted is a caller-registered eviction callback (e.g. the ICE cache
	// bumps sequence numbers here). We invoke it from our own handler so wrapping
	// a cache never steals the caller's OnEvicted hook. Like go-cache's own
	// OnEvicted, it is expected to be set once at construction, before the cache
	// is shared across goroutines, so it needs no synchronization.
	userOnEvicted func(string, interface{})
}

// New returns an instrumented cache. name is used as the "name" label on every
// metric and MUST be stable and low-cardinality (e.g. "aws.ssm"); never derive
// it from a cache key or other unbounded value. defaultExpiration and
// cleanupInterval are passed through to cache.New unchanged.
func New(name string, defaultExpiration, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		Cache: cache.New(defaultExpiration, cleanupInterval),
		name:  name,
	}
	c.Cache.OnEvicted(c.onEvicted)
	c.initMetrics()
	return c
}

// initMetrics pre-touches every counter/gauge series so they report 0 from
// process start rather than blinking into existence on first use. rate() and
// hit-ratio queries then behave from t0 and never read as "no data" mid-incident.
// The flush_size histogram is intentionally left lazy: observing a fake 0 would
// corrupt the size distribution, and flushes are rare, so the series appears on
// the first real flush.
func (c *Cache) initMetrics() {
	getsTotal.Add(0, c.labels(MetricLabelResult, ResultHit))
	getsTotal.Add(0, c.labels(MetricLabelResult, ResultMiss))
	addsTotal.Add(0, c.labels(MetricLabelResult, ResultAdded))
	addsTotal.Add(0, c.labels(MetricLabelResult, ResultExists))
	evictionsTotal.Add(0, c.labels())
	flushesTotal.Add(0, c.labels())
	entries.Set(0, c.labels())
}

// labels builds the metric label set for this cache: always the "name" label,
// plus any extra key/value pairs supplied as a flat, even-length list.
func (c *Cache) labels(kv ...string) map[string]string {
	l := map[string]string{MetricLabelName: c.name}
	for i := 0; i+1 < len(kv); i += 2 {
		l[kv[i]] = kv[i+1]
	}
	return l
}

// Get records a hit or miss and delegates to the underlying cache.
func (c *Cache) Get(k string) (interface{}, bool) {
	v, ok := c.Cache.Get(k)
	if ok {
		getsTotal.Inc(c.labels(MetricLabelResult, ResultHit))
	} else {
		getsTotal.Inc(c.labels(MetricLabelResult, ResultMiss))
	}
	return v, ok
}

// Add records whether the key was added or already existed (a suppressed
// duplicate) and delegates to the underlying cache.
func (c *Cache) Add(k string, x interface{}, d time.Duration) error {
	err := c.Cache.Add(k, x, d)
	if err != nil {
		addsTotal.Inc(c.labels(MetricLabelResult, ResultExists))
		return err
	}
	addsTotal.Inc(c.labels(MetricLabelResult, ResultAdded))
	c.updateEntries()
	return nil
}

// Set delegates to the underlying cache and refreshes the entries gauge.
func (c *Cache) Set(k string, x interface{}, d time.Duration) {
	c.Cache.Set(k, x, d)
	c.updateEntries()
}

// SetDefault delegates to the underlying cache and refreshes the entries gauge.
func (c *Cache) SetDefault(k string, x interface{}) {
	c.Cache.SetDefault(k, x)
	c.updateEntries()
}

// Delete removes a single entry. If the entry existed, the removal is counted by
// evictions_total via the eviction callback (see onEvicted); go-cache does not
// distinguish an explicit delete from a TTL expiry, and neither do we.
func (c *Cache) Delete(k string) {
	c.Cache.Delete(k)
	c.updateEntries()
}

// Flush empties the cache. go-cache's Flush does not fire OnEvicted, so we
// account for it here as a flush event plus the number of entries discarded (as
// a histogram), and reset the entries gauge. flush_size's _sum gives the total
// number of entries discarded by flushes over time.
func (c *Cache) Flush() {
	n := c.Cache.ItemCount()
	c.Cache.Flush()
	flushesTotal.Inc(c.labels())
	flushSize.Observe(float64(n), c.labels())
	entries.Set(0, c.labels())
}

// OnEvicted registers a caller eviction callback, invoked from the wrapper's own
// handler so instrumentation and caller bookkeeping coexist. Like go-cache's
// OnEvicted, call this once at construction before the cache is shared across
// goroutines.
func (c *Cache) OnEvicted(f func(string, interface{})) {
	c.userOnEvicted = f
}

// onEvicted is registered with the underlying cache and fires for every real
// per-key removal — both explicit Delete and TTL expiry. It records the removal,
// refreshes the entries gauge, then invokes the caller's callback if set.
func (c *Cache) onEvicted(k string, v interface{}) {
	evictionsTotal.Inc(c.labels())
	c.updateEntries()
	if c.userOnEvicted != nil {
		c.userOnEvicted(k, v)
	}
}

// updateEntries refreshes the size gauge. ItemCount is O(1) under the cache's
// read lock, and this runs off the scrape path, so scrapes never trigger a
// cache-wide walk or block on cache mutation.
func (c *Cache) updateEntries() {
	entries.Set(float64(c.Cache.ItemCount()), c.labels())
}
