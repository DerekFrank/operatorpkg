package cache_test

import (
	"testing"
	"time"

	opcache "github.com/awslabs/operatorpkg/cache"
	. "github.com/awslabs/operatorpkg/test/expectations"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache")
}

// value reads a counter's current value for this cache's name plus extra labels,
// returning 0 when the series does not yet exist.
func value(metric string, name string, kv ...string) float64 {
	labels := map[string]string{opcache.MetricLabelName: name}
	for i := 0; i+1 < len(kv); i += 2 {
		labels[kv[i]] = kv[i+1]
	}
	m := GetMetric("operator_cache_"+metric, labels)
	if m == nil {
		return 0
	}
	if c := m.GetCounter(); c != nil {
		return c.GetValue()
	}
	if g := m.GetGauge(); g != nil {
		return g.GetValue()
	}
	return 0
}

var _ = Describe("Cache", func() {
	var name string
	var c *opcache.Cache

	BeforeEach(func() {
		// Unique name per spec so metric series don't bleed across tests.
		name = "test-" + CurrentSpecReport().LeafNodeText
		c = opcache.New(name, time.Hour, 0) // cleanupInterval 0 => no janitor
	})

	It("initializes counter and gauge series to zero", func() {
		Expect(value("gets_total", name, opcache.MetricLabelResult, opcache.ResultHit)).To(BeZero())
		Expect(value("gets_total", name, opcache.MetricLabelResult, opcache.ResultMiss)).To(BeZero())
		Expect(value("adds_total", name, opcache.MetricLabelResult, opcache.ResultAdded)).To(BeZero())
		Expect(value("adds_total", name, opcache.MetricLabelResult, opcache.ResultExists)).To(BeZero())
		Expect(value("evictions_total", name)).To(BeZero())
		Expect(value("flushes_total", name)).To(BeZero())
		Expect(value("entries", name)).To(BeZero())
	})

	It("records hits and misses on Get", func() {
		c.SetDefault("k", "v")
		_, ok := c.Get("k")
		Expect(ok).To(BeTrue())
		_, ok = c.Get("missing")
		Expect(ok).To(BeFalse())

		Expect(value("gets_total", name, opcache.MetricLabelResult, opcache.ResultHit)).To(BeEquivalentTo(1))
		Expect(value("gets_total", name, opcache.MetricLabelResult, opcache.ResultMiss)).To(BeEquivalentTo(1))
	})

	It("records add vs duplicate-suppressed on Add", func() {
		Expect(c.Add("k", "v", 0)).To(Succeed())
		Expect(c.Add("k", "v2", 0)).ToNot(Succeed()) // duplicate => suppressed

		Expect(value("adds_total", name, opcache.MetricLabelResult, opcache.ResultAdded)).To(BeEquivalentTo(1))
		Expect(value("adds_total", name, opcache.MetricLabelResult, opcache.ResultExists)).To(BeEquivalentTo(1))
	})

	It("counts an explicit Delete as an eviction", func() {
		c.SetDefault("k", "v")
		c.Delete("k")

		Expect(value("evictions_total", name)).To(BeEquivalentTo(1))
	})

	It("does not count a Delete of a missing key", func() {
		c.Delete("missing")
		Expect(value("evictions_total", name)).To(BeZero())
	})

	It("counts a TTL expiry as an eviction", func() {
		// Short TTL with a running janitor so the entry expires and is swept.
		c = opcache.New(name, 10*time.Millisecond, 10*time.Millisecond)
		c.SetDefault("k", "v")
		Eventually(func() float64 { return value("evictions_total", name) }, time.Second).Should(BeEquivalentTo(1))
	})

	It("accounts a Flush as an event and a size observation, separate from evictions", func() {
		c.SetDefault("a", 1)
		c.SetDefault("b", 2)
		c.SetDefault("cc", 3)
		c.Flush()

		Expect(value("flushes_total", name)).To(BeEquivalentTo(1))
		// Flush does not fire per-key OnEvicted, so evictions_total is untouched.
		Expect(value("evictions_total", name)).To(BeZero())
		Expect(value("entries", name)).To(BeZero())
	})

	It("tracks the entries gauge across mutations", func() {
		c.SetDefault("a", 1)
		c.SetDefault("b", 2)
		Expect(value("entries", name)).To(BeEquivalentTo(2))
		c.Delete("a")
		Expect(value("entries", name)).To(BeEquivalentTo(1))
	})

	It("still invokes a caller-registered OnEvicted callback", func() {
		evicted := map[string]interface{}{}
		c.OnEvicted(func(k string, v interface{}) { evicted[k] = v })
		c.SetDefault("k", "v")
		c.Delete("k")

		Expect(evicted).To(HaveKeyWithValue("k", "v"))
		// And the metric is still recorded alongside the caller callback.
		Expect(value("evictions_total", name)).To(BeEquivalentTo(1))
	})
})
