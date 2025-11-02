package persist

import (
	"sync"
	"sync/atomic"
)

// CompoundPrimarySyncMap 泛型并发安全Map
// K: 键类型（必须可比较）
// V: 值类型
type CompoundPrimarySyncMap[V any] struct {
	mu     sync.Mutex
	read   atomic.Pointer[readCompoundOnly[V]]              // read 包含可以并发访问的map部分
	dirty  map[GlobalKeyTypeHashCompoundPrimaryId]*entry[V] // dirty 包含需要持有mu才能访问的map部分
	misses int                                              // misses 记录自上次read更新以来需要锁定mu的加载次数
}

// readCompoundOnly 是map的只读版本
type readCompoundOnly[V any] struct {
	m       map[GlobalKeyTypeHashCompoundPrimaryId]*entry[V]
	amended bool // 如果dirty map包含一些不在m中的键，则为true
}

// loadreadCompoundOnly 加载只读map
func (m *CompoundPrimarySyncMap[V]) loadreadCompoundOnly() readCompoundOnly[V] {
	if p := m.read.Load(); p != nil {
		return *p
	}
	return readCompoundOnly[V]{}
}

// Load 返回存储在map中的键对应的值，如果不存在则返回零值
func (m *CompoundPrimarySyncMap[V]) Load(key GlobalKeyTypeHashCompoundPrimaryId) (value V, ok bool) {
	read := m.loadreadCompoundOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadreadCompoundOnly()
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
func (m *CompoundPrimarySyncMap[V]) Store(key GlobalKeyTypeHashCompoundPrimaryId, value V) {
	_, _ = m.Swap(key, value)
}

// Clear 删除所有条目，生成空Map
func (m *CompoundPrimarySyncMap[V]) Clear() {
	read := m.loadreadCompoundOnly()
	if len(read.m) == 0 && !read.amended {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	read = m.loadreadCompoundOnly()
	if len(read.m) > 0 || read.amended {
		m.read.Store(&readCompoundOnly[V]{})
	}

	clear(m.dirty)
	m.misses = 0
}

// LoadOrStore 返回键的现有值(如果存在)
// 否则，它存储并返回给定的值
// loaded结果为true表示值已加载，false表示已存储
func (m *CompoundPrimarySyncMap[V]) LoadOrStore(key GlobalKeyTypeHashCompoundPrimaryId, value V) (actual V, loaded bool) {
	read := m.loadreadCompoundOnly()
	if e, ok := read.m[key]; ok {
		actual, loaded, ok := e.tryLoadOrStore(value)
		if ok {
			return actual, loaded
		}
	}

	m.mu.Lock()
	read = m.loadreadCompoundOnly()
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
			m.read.Store(&readCompoundOnly[V]{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry[V](value)
		actual, loaded = value, false
	}
	m.mu.Unlock()

	return actual, loaded
}

// LoadAndDelete 删除键的值，返回之前的值(如果有), loaded结果报告键是否存在
func (m *CompoundPrimarySyncMap[V]) LoadAndDelete(key GlobalKeyTypeHashCompoundPrimaryId) (value V, loaded bool) {
	read := m.loadreadCompoundOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadreadCompoundOnly()
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
func (m *CompoundPrimarySyncMap[V]) Delete(key GlobalKeyTypeHashCompoundPrimaryId) {
	m.LoadAndDelete(key)
}

// Swap 交换键的值并返回之前的值(如果有), loaded结果报告键是否存在
func (m *CompoundPrimarySyncMap[V]) Swap(key GlobalKeyTypeHashCompoundPrimaryId, value V) (previous V, loaded bool) {
	read := m.loadreadCompoundOnly()
	if e, ok := read.m[key]; ok {
		if v, ok := e.trySwap(&value); ok {
			if v == nil {
				return previous, false
			}
			return *v, true
		}
	}

	m.mu.Lock()
	read = m.loadreadCompoundOnly()
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
			m.read.Store(&readCompoundOnly[V]{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry[V](value)
	}
	m.mu.Unlock()
	return previous, loaded
}

// CompareAndSwap 如果存储在map中的值等于old，则交换old和new值
func (m *CompoundPrimarySyncMap[V]) CompareAndSwap(key GlobalKeyTypeHashCompoundPrimaryId, old, new V) (swapped bool) {
	read := m.loadreadCompoundOnly()
	if e, ok := read.m[key]; ok {
		return e.tryCompareAndSwap(old, new)
	} else if !read.amended {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	read = m.loadreadCompoundOnly()
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
func (m *CompoundPrimarySyncMap[V]) CompareAndDelete(key GlobalKeyTypeHashCompoundPrimaryId, old V) (deleted bool) {
	read := m.loadreadCompoundOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadreadCompoundOnly()
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
func (m *CompoundPrimarySyncMap[V]) Range(f func(key GlobalKeyTypeHashCompoundPrimaryId, value V) bool) {
	read := m.loadreadCompoundOnly()
	if read.amended {
		m.mu.Lock()
		read = m.loadreadCompoundOnly()
		if read.amended {
			read = readCompoundOnly[V]{m: m.dirty}
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
func (m *CompoundPrimarySyncMap[V]) missLocked() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	m.read.Store(&readCompoundOnly[V]{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

// dirtyLocked 初始化dirty map
func (m *CompoundPrimarySyncMap[V]) dirtyLocked() {
	if m.dirty != nil {
		return
	}

	read := m.loadreadCompoundOnly()
	m.dirty = make(map[GlobalKeyTypeHashCompoundPrimaryId]*entry[V], len(read.m))
	for k, e := range read.m {
		if !e.tryExpungeLocked() {
			m.dirty[k] = e
		}
	}
}
