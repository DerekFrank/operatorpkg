package metrics

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

// keyIndex records the label sets a metric has emitted, grouped by the values of its
// Key dimensions, so that DeleteByKey can delete a key's series directly. A metric
// without Key dimensions has no index.
type keyIndex struct {
	keys, names []string
	// mu orders each change to the vec with the index update for it. Without it, a
	// DeleteByKey racing an emit could forget the label set before the emit creates its
	// series, leaving a series no DeleteByKey would find.
	mu sync.Mutex
	// series maps the Key values, then all the values, to the labels emitted with them.
	series map[string]map[string]prometheus.Labels
}

func newKeyIndex(labels []Label) *keyIndex {
	k := &keyIndex{series: map[string]map[string]prometheus.Labels{}}
	for _, l := range labels {
		k.names = append(k.names, l.Name)
		if l.Key {
			k.keys = append(k.keys, l.Name)
		}
	}
	if len(k.keys) == 0 {
		return nil
	}
	return k
}

// join appends the values of names in labels to buf, each followed by a separator so
// that ("ab", "c") and ("a", "bc") differ.
func join(buf []byte, names []string, labels map[string]string) []byte {
	for _, name := range names {
		buf = append(buf, labels[name]...)
		buf = append(buf, model.SeparatorByte)
	}
	return buf
}

// emit runs f, which creates or updates the series for labels, and records labels.
func (k *keyIndex) emit(labels map[string]string, f func()) {
	if k == nil {
		f()
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	f()
	var kb, lb [256]byte
	key, all := join(kb[:0], k.keys, labels), join(lb[:0], k.names, labels)
	if _, ok := k.series[string(key)][string(all)]; ok {
		return
	}
	if k.series[string(key)] == nil {
		k.series[string(key)] = map[string]prometheus.Labels{}
	}
	k.series[string(key)][string(all)] = maps.Clone(labels)
}

// delete runs f, which deletes the series for labels, and forgets labels.
func (k *keyIndex) delete(labels map[string]string, f func()) {
	if k == nil {
		f()
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	f()
	var kb, lb [256]byte
	key, all := join(kb[:0], k.keys, labels), join(lb[:0], k.names, labels)
	delete(k.series[string(key)], string(all))
	if len(k.series[string(key)]) == 0 {
		delete(k.series, string(key))
	}
}

// deleteByKey calls del for every label set emitted with key, which must give exactly
// the Key dimensions, and forgets them.
func (k *keyIndex) deleteByKey(key map[string]string, del func(prometheus.Labels) bool) {
	k.mustBeKey(key)
	k.mu.Lock()
	defer k.mu.Unlock()
	var kb [256]byte
	b := join(kb[:0], k.keys, key)
	for _, l := range k.series[string(b)] {
		del(l)
	}
	delete(k.series, string(b))
}

// replace calls set for each of series, which must all have the Key values in key,
// then del for every other label set emitted with key, and records series as key's
// label sets. Setting before deleting means a scrape never finds key without a series.
func (k *keyIndex) replace(key map[string]string, series []Sample, set func(Sample), del func(prometheus.Labels) bool) {
	k.mustBeKey(key)
	k.mu.Lock()
	defer k.mu.Unlock()
	var kb, sb, lb [256]byte
	b := join(kb[:0], k.keys, key)
	old := k.series[string(b)]
	next := make(map[string]prometheus.Labels, len(series))
	for _, s := range series {
		if string(join(sb[:0], k.keys, s.Labels)) != string(b) {
			panic(fmt.Sprintf("ReplaceByKey series %v doesn't have the Key values %v", s.Labels, key))
		}
		set(s)
		all := join(lb[:0], k.names, s.Labels)
		if l, ok := old[string(all)]; ok {
			next[string(all)] = l
		} else if _, ok := next[string(all)]; !ok {
			next[string(all)] = maps.Clone(s.Labels)
		}
	}
	for all, l := range old {
		if _, ok := next[all]; !ok {
			del(l)
		}
	}
	if len(next) > 0 {
		k.series[string(b)] = next
	} else {
		delete(k.series, string(b))
	}
}

func (k *keyIndex) mustBeKey(key map[string]string) {
	if k == nil || len(key) != len(k.keys) || slices.ContainsFunc(k.keys, func(name string) bool { _, ok := key[name]; return !ok }) {
		panic(fmt.Sprintf("%v must give exactly the metric's Key labels", key))
	}
}

// deletePartialMatch runs f, the client's DeletePartialMatch, and forgets the label sets
// it deleted.
func (k *keyIndex) deletePartialMatch(partial map[string]string, f func()) {
	if k == nil {
		f()
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	f()
	for key, series := range k.series {
		maps.DeleteFunc(series, func(_ string, l prometheus.Labels) bool {
			for name, v := range partial {
				if lv, ok := l[name]; !ok || lv != v {
					return false
				}
			}
			return true
		})
		if len(series) == 0 {
			delete(k.series, key)
		}
	}
}

func (k *keyIndex) reset(f func()) {
	if k == nil {
		f()
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	f()
	clear(k.series)
}
