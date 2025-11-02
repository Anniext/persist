// Package data 提供泛型哈希索引
package data

// HashIndex 泛型哈希索引
// K: 键类型（必须可比较）
// V: 值类型
type HashIndex[K comparable, V any] struct {
	*SyncMap[K, V]
}

// NewHashIndex 创建新的哈希索引
func NewHashIndex[K comparable, V any]() *HashIndex[K, V] {
	return &HashIndex[K, V]{
		SyncMap: &SyncMap[K, V]{},
	}
}

// Get 获取值
func (h *HashIndex[K, V]) Get(key K) (V, bool) {
	return h.Load(key)
}

// Set 设置值
func (h *HashIndex[K, V]) Set(key K, value V) {
	h.Store(key, value)
}

// Remove 删除值
func (h *HashIndex[K, V]) Remove(key K) {
	h.Delete(key)
}

// Has 检查键是否存在
func (h *HashIndex[K, V]) Has(key K) bool {
	_, ok := h.Load(key)
	return ok
}

// MultiHashIndex 泛型多值哈希索引（一个键对应多个值）
// K: 键类型（必须可比较）
// V: 值类型（必须可比较）
type MultiHashIndex[K comparable, V comparable] struct {
	*SyncMap[K, *SyncMap[V, bool]]
}

// NewMultiHashIndex 创建新的多值哈希索引
func NewMultiHashIndex[K comparable, V comparable]() *MultiHashIndex[K, V] {
	return &MultiHashIndex[K, V]{
		SyncMap: &SyncMap[K, *SyncMap[V, bool]]{},
	}
}

// Add 添加值到键
func (h *MultiHashIndex[K, V]) Add(key K, value V) {
	set, ok := h.Load(key)
	if !ok {
		set = &SyncMap[V, bool]{}
		actual, loaded := h.LoadOrStore(key, set)
		if loaded {
			set = actual
		}
	}
	set.Store(value, true)
}

// Remove 从键中删除值
func (h *MultiHashIndex[K, V]) Remove(key K, value V) {
	set, ok := h.Load(key)
	if ok {
		set.Delete(value)
	}
}

// RemoveAll 删除键的所有值
func (h *MultiHashIndex[K, V]) RemoveAll(key K) {
	h.Delete(key)
}

// Has 检查键是否包含值
func (h *MultiHashIndex[K, V]) Has(key K, value V) bool {
	set, ok := h.Load(key)
	if !ok {
		return false
	}
	_, exists := set.Load(value)
	return exists
}

// GetAll 获取键的所有值
func (h *MultiHashIndex[K, V]) GetAll(key K) []V {
	set, ok := h.Load(key)
	if !ok {
		return nil
	}

	values := make([]V, 0)
	set.Range(func(value V, _ bool) bool {
		values = append(values, value)
		return true
	})
	return values
}

// RangeValues 遍历键的所有值
func (h *MultiHashIndex[K, V]) RangeValues(key K, f func(value V) bool) {
	set, ok := h.Load(key)
	if !ok {
		return
	}

	set.Range(func(value V, _ bool) bool {
		return f(value)
	})
}
