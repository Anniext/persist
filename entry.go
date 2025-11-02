package persist

import "sync/atomic"

// entry 是map中对应特定键的槽位
type entry[V any] struct {
	p atomic.Pointer[V] // p 指向为entry存储的值
}

// load 从entry加载值
func (e *entry[V]) load() (value V, ok bool) {
	p := e.p.Load()
	if p == nil || p == expunged[V]() {
		return value, false
	}
	return *p, true
}

// tryCompareAndSwap 比较entry与给定的旧值，如果相等则交换为新值
func (e *entry[V]) tryCompareAndSwap(old, new V) bool {
	p := e.p.Load()
	// 注意：这里无法直接比较泛型值，需要使用any进行比较
	if p == nil || p == expunged[V]() || any(*p) != any(old) {
		return false
	}

	nc := new
	for {
		if e.p.CompareAndSwap(p, &nc) {
			return true
		}
		p = e.p.Load()
		if p == nil || expunged[V]() == p || any(*p) != any(old) {
			return false
		}
	}
}

// unexpungeLocked 确保entry未标记为expunged
func (e *entry[V]) unexpungeLocked() (wasExpunged bool) {
	return e.p.CompareAndSwap(expunged[V](), nil)
}

// swapLocked 无条件地将值交换到entry中
func (e *entry[V]) swapLocked(i *V) *V {
	return e.p.Swap(i)
}

// tryLoadOrStore 原子地加载或存储值（如果entry未被expunged）
func (e *entry[V]) tryLoadOrStore(i V) (actual V, loaded, ok bool) {
	p := e.p.Load()
	if p == expunged[V]() {
		return actual, false, false
	}
	if p != nil {
		return *p, true, true
	}

	ic := i
	for {
		if e.p.CompareAndSwap(nil, &ic) {
			return i, false, true
		}
		p = e.p.Load()
		if p == expunged[V]() {
			return actual, false, false
		}
		if p != nil {
			return *p, true, true
		}
	}
}

// delete 从entry删除值
func (e *entry[V]) delete() (value V, ok bool) {
	for {
		p := e.p.Load()
		if p == nil || p == expunged[V]() {
			return value, false
		}
		if e.p.CompareAndSwap(p, nil) {
			return *p, true
		}
	}
}

// trySwap 如果entry未被expunged则交换值
func (e *entry[V]) trySwap(i *V) (*V, bool) {
	for {
		p := e.p.Load()
		if p == expunged[V]() {
			return nil, false
		}
		if e.p.CompareAndSwap(p, i) {
			return p, true
		}
	}
}

// tryExpungeLocked 尝试标记entry为expunged
func (e *entry[V]) tryExpungeLocked() (isExpunged bool) {
	p := e.p.Load()
	for p == nil {
		if e.p.CompareAndSwap(nil, expunged[V]()) {
			return true
		}
		p = e.p.Load()
	}
	return p == expunged[V]()
}

// newEntry 创建新的entry
func newEntry[V any](i V) *entry[V] {
	e := &entry[V]{}
	e.p.Store(&i)
	return e
}
