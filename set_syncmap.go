package persist

import (
	"sync"
	"sync/atomic"
)

// SetSyncMap 泛型并发安全Map
// K: 键类型（必须可比较）
// V: 值类型
type SetSyncMap[V comparable] struct {
	mu     sync.Mutex
	read   atomic.Pointer[readSetOnly[V]] // read 包含可以并发访问的map部分
	dirty  map[V]*setEntry                // dirty 包含需要持有mu才能访问的map部分
	misses int                            // misses 记录自上次read更新以来需要锁定mu的加载次数
}

// readSetOnly 是map的只读版本
type readSetOnly[V comparable] struct {
	m       map[V]*setEntry
	amended bool // 如果dirty map包含一些不在m中的键，则为true
}

func setExpunged() *bool {
	return new(bool)
}

// setEntry 是map中对应特定键的槽位
type setEntry struct {
	p atomic.Pointer[bool] // p 指向为entry存储的值
}

// newEntry 创建新的entry
func newSetEntry(i bool) *setEntry {
	e := &setEntry{}
	e.p.Store(&i)
	return e
}

// loadReadOnly 加载只读map
func (m *SetSyncMap[V]) loadReadOnly() readSetOnly[V] {
	if p := m.read.Load(); p != nil {
		return *p
	}
	return readSetOnly[V]{}
}

// Load 返回存储在map中的键对应的值，如果不存在则返回零值, ok结果指示是否在map中找到了值
func (m *SetSyncMap[V]) Load(key V) (value bool, ok bool) {
	read := m.loadReadOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadReadOnly()
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if !ok {
		return value, false
	}
	return e.load()
}

// load 从entry加载值
func (e *setEntry) load() (value bool, ok bool) {
	p := e.p.Load()
	if p == nil || p == setExpunged() {
		return value, false
	}
	return *p, true
}

// Store 设置键的值
func (m *SetSyncMap[V]) Store(key V, value bool) {
	_, _ = m.Swap(key, value)
}

// Clear 删除所有条目，生成空Map
func (m *SetSyncMap[V]) Clear() {
	read := m.loadReadOnly()
	if len(read.m) == 0 && !read.amended {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	read = m.loadReadOnly()
	if len(read.m) > 0 || read.amended {
		m.read.Store(&readSetOnly[V]{})
	}

	clear(m.dirty)
	m.misses = 0
}

// tryCompareAndSwap 比较entry与给定的旧值，如果相等则交换为新值
func (e *setEntry) tryCompareAndSwap(old, new bool) bool {
	p := e.p.Load()
	// 注意：这里无法直接比较泛型值，需要使用any进行比较
	if p == nil || p == setExpunged() || *p != old {
		return false
	}

	nc := new
	for {
		if e.p.CompareAndSwap(p, &nc) {
			return true
		}
		p = e.p.Load()
		if p == nil || setExpunged() == p || *p != old {
			return false
		}
	}
}

// unexpungeLocked 确保entry未标记为expunged
func (e *setEntry) unexpungeLocked() (wasExpunged bool) {
	return e.p.CompareAndSwap(setExpunged(), nil)
}

// swapLocked 无条件地将值交换到entry中
func (e *setEntry) swapLocked(i *bool) *bool {
	return e.p.Swap(i)
}

// LoadOrStore 返回键的现有值(如果存在), 否则，它存储并返回给定的值, loaded结果为true表示值已加载，false表示已存储
func (m *SetSyncMap[V]) LoadOrStore(key V, value bool) (actual bool, loaded bool) {
	read := m.loadReadOnly()
	if e, ok := read.m[key]; ok {
		actual, loaded, ok := e.tryLoadOrStore(value)
		if ok {
			return actual, loaded
		}
	}

	m.mu.Lock()
	read = m.loadReadOnly()
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			m.dirty[key] = e
		}
		actual, loaded, _ = e.tryLoadOrStore(value)
	} else if e, ok := m.dirty[key]; ok {
		actual, loaded, _ = e.tryLoadOrStore(value)
		m.missLocked()
	} else {
		if !read.amended {
			m.dirtyLocked()
			m.read.Store(&readSetOnly[V]{m: read.m, amended: true})
		}
		m.dirty[key] = newSetEntry[V](value)
		actual, loaded = value, false
	}
	m.mu.Unlock()

	return actual, loaded
}

// tryLoadOrStore 原子地加载或存储值（如果entry未被expunged）
func (e *setEntry) tryLoadOrStore(i bool) (actual bool, loaded, ok bool) {
	p := e.p.Load()
	if p == setExpunged() {
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
		if p == setExpunged() {
			return actual, false, false
		}
		if p != nil {
			return *p, true, true
		}
	}
}

// LoadAndDelete 删除键的值，返回之前的值(如果有), loaded结果报告键是否存在
func (m *SetSyncMap[V]) LoadAndDelete(key V) (value bool, loaded bool) {
	read := m.loadReadOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadReadOnly()
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			delete(m.dirty, key)
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if ok {
		return e.delete()
	}
	return value, false
}

// Delete 删除键的值
func (m *SetSyncMap[V]) Delete(key V) {
	m.LoadAndDelete(key)
}

// delete 从entry删除值
func (e *setEntry) delete() (value bool, ok bool) {
	for {
		p := e.p.Load()
		if p == nil || p == setExpunged() {
			return value, false
		}
		if e.p.CompareAndSwap(p, nil) {
			return *p, true
		}
	}
}

// trySwap 如果entry未被expunged则交换值
func (e *setEntry) trySwap(i *bool) (*bool, bool) {
	for {
		p := e.p.Load()
		if p == setExpunged() {
			return nil, false
		}
		if e.p.CompareAndSwap(p, i) {
			return p, true
		}
	}
}

// Swap 交换键的值并返回之前的值(如果有), loaded结果报告键是否存在
func (m *SetSyncMap[V]) Swap(key V, value bool) (previous bool, loaded bool) {
	read := m.loadReadOnly()
	if e, ok := read.m[key]; ok {
		if v, ok := e.trySwap(&value); ok {
			if v == nil {
				return previous, false
			}
			return *v, true
		}
	}

	m.mu.Lock()
	read = m.loadReadOnly()
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			m.dirty[key] = e
		}
		if v := e.swapLocked(&value); v != nil {
			loaded = true
			previous = *v
		}
	} else if e, ok := m.dirty[key]; ok {
		if v := e.swapLocked(&value); v != nil {
			loaded = true
			previous = *v
		}
	} else {
		if !read.amended {
			m.dirtyLocked()
			m.read.Store(&readSetOnly[V]{m: read.m, amended: true})
		}
		m.dirty[key] = newSetEntry(value)
	}
	m.mu.Unlock()
	return previous, loaded
}

// CompareAndSwap 如果存储在map中的值等于old，则交换old和new值
func (m *SetSyncMap[V]) CompareAndSwap(key V, old, new bool) (swapped bool) {
	read := m.loadReadOnly()
	if e, ok := read.m[key]; ok {
		return e.tryCompareAndSwap(old, new)
	} else if !read.amended {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	read = m.loadReadOnly()
	swapped = false
	if e, ok := read.m[key]; ok {
		swapped = e.tryCompareAndSwap(old, new)
	} else if e, ok := m.dirty[key]; ok {
		swapped = e.tryCompareAndSwap(old, new)
		m.missLocked()
	}
	return swapped
}

// CompareAndDelete 如果值等于old则删除键的条目
func (m *SetSyncMap[V]) CompareAndDelete(key V, old bool) (deleted bool) {
	read := m.loadReadOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadReadOnly()
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			m.missLocked()
		}
		m.mu.Unlock()
	}
	for ok {
		p := e.p.Load()
		if p == nil || p == setExpunged() || *p != old {
			return false
		}
		if e.p.CompareAndSwap(p, nil) {
			return true
		}
	}
	return false
}

// Range 对map中存在的每个键和值依次调用f, 如果f返回false，range停止迭代
func (m *SetSyncMap[V]) Range(f func(key V, value bool) bool) {
	read := m.loadReadOnly()
	if read.amended {
		m.mu.Lock()
		read = m.loadReadOnly()
		if read.amended {
			read = readSetOnly[V]{m: m.dirty}
			copyRead := read
			m.read.Store(&copyRead)
			m.dirty = nil
			m.misses = 0
		}
		m.mu.Unlock()
	}

	for k, e := range read.m {
		v, ok := e.load()
		if !ok {
			continue
		}
		if !f(k, v) {
			break
		}
	}
}

// missLocked 记录miss并可能提升dirty map
func (m *SetSyncMap[V]) missLocked() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	m.read.Store(&readSetOnly[V]{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

// dirtyLocked 初始化dirty map
func (m *SetSyncMap[V]) dirtyLocked() {
	if m.dirty != nil {
		return
	}

	read := m.loadReadOnly()
	m.dirty = make(map[V]*setEntry, len(read.m))
	for k, e := range read.m {
		if !e.tryExpungeLocked() {
			m.dirty[k] = e
		}
	}
}

// tryExpungeLocked 尝试标记entry为expunged
func (e *setEntry) tryExpungeLocked() (isExpunged bool) {
	p := e.p.Load()
	for p == nil {
		if e.p.CompareAndSwap(nil, setExpunged()) {
			return true
		}
		p = e.p.Load()
	}
	return p == setExpunged()
}
