package persist

import (
	"sync"
	"sync/atomic"
)

// PrimarySyncMap 泛型并发安全Map
// K: 键类型（必须可比较）
// V: 值类型
type PrimarySyncMap[V GlobalModel] struct {
	mu     sync.Mutex
	read   atomic.Pointer[readOnly[V]]              // read 包含可以并发访问的map部分
	dirty  map[GlobalKeyTypeHashPrimaryId]*entry[V] // dirty 包含需要持有mu才能访问的map部分
	misses int                                      // misses 记录自上次read更新以来需要锁定mu的加载次数
}

// readOnly 是map的只读版本
type readOnly[V GlobalModel] struct {
	m       map[GlobalKeyTypeHashPrimaryId]*entry[V]
	amended bool // 如果dirty map包含一些不在m中的键，则为true
}

// expunged 是一个任意指针，标记已从dirty map中删除的条目
func expunged[V any]() *V {
	return new(V)
}

// loadReadOnly 加载只读map
func (m *PrimarySyncMap[V]) loadReadOnly() readOnly[V] {
	if p := m.read.Load(); p != nil {
		return *p
	}
	return readOnly[V]{}
}

// Load 返回存储在map中的键对应的值，如果不存在则返回零值, ok结果指示是否在map中找到了值
func (m *PrimarySyncMap[V]) Load(key GlobalKeyTypeHashPrimaryId) (value V, ok bool) {
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

// Store 设置键的值
func (m *PrimarySyncMap[V]) Store(key GlobalKeyTypeHashPrimaryId, value V) {
	_, _ = m.Swap(key, value)
}

// Clear 删除所有条目，生成空Map
func (m *PrimarySyncMap[V]) Clear() {
	read := m.loadReadOnly()
	if len(read.m) == 0 && !read.amended {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	read = m.loadReadOnly()
	if len(read.m) > 0 || read.amended {
		m.read.Store(&readOnly[V]{})
	}

	clear(m.dirty)
	m.misses = 0
}

// LoadOrStore 返回键的现有值(如果存在), 否则，它存储并返回给定的值, loaded结果为true表示值已加载，false表示已存储
func (m *PrimarySyncMap[V]) LoadOrStore(key GlobalKeyTypeHashPrimaryId, value V) (actual V, loaded bool) {
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
			m.read.Store(&readOnly[V]{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry[V](value)
		actual, loaded = value, false
	}
	m.mu.Unlock()

	return actual, loaded
}

// LoadAndDelete 删除键的值，返回之前的值(如果有), loaded结果报告键是否存在
func (m *PrimarySyncMap[V]) LoadAndDelete(key GlobalKeyTypeHashPrimaryId) (value V, loaded bool) {
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
func (m *PrimarySyncMap[V]) Delete(key GlobalKeyTypeHashPrimaryId) {
	m.LoadAndDelete(key)
}

// Swap 交换键的值并返回之前的值(如果有), loaded结果报告键是否存在
func (m *PrimarySyncMap[V]) Swap(key GlobalKeyTypeHashPrimaryId, value V) (previous V, loaded bool) {
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
			m.read.Store(&readOnly[V]{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry[V](value)
	}
	m.mu.Unlock()
	return previous, loaded
}

// CompareAndSwap 如果存储在map中的值等于old，则交换old和new值
func (m *PrimarySyncMap[V]) CompareAndSwap(key GlobalKeyTypeHashPrimaryId, old, new V) (swapped bool) {
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
func (m *PrimarySyncMap[V]) CompareAndDelete(key GlobalKeyTypeHashPrimaryId, old V) (deleted bool) {
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
		if p == nil || p == expunged[V]() || any(*p) != any(old) {
			return false
		}
		if e.p.CompareAndSwap(p, nil) {
			return true
		}
	}
	return false
}

// Range 对map中存在的每个键和值依次调用f, 如果f返回false，range停止迭代
func (m *PrimarySyncMap[V]) Range(f func(key GlobalKeyTypeHashPrimaryId, value V) bool) {
	read := m.loadReadOnly()
	if read.amended {
		m.mu.Lock()
		read = m.loadReadOnly()
		if read.amended {
			read = readOnly[V]{m: m.dirty}
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
func (m *PrimarySyncMap[V]) missLocked() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	m.read.Store(&readOnly[V]{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

// dirtyLocked 初始化dirty map
func (m *PrimarySyncMap[V]) dirtyLocked() {
	if m.dirty != nil {
		return
	}

	read := m.loadReadOnly()
	m.dirty = make(map[GlobalKeyTypeHashPrimaryId]*entry[V], len(read.m))
	for k, e := range read.m {
		if !e.tryExpungeLocked() {
			m.dirty[k] = e
		}
	}
}
