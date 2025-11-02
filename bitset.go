package persist

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"sync"
)

var cache sync.Map // map[reflect.Type]*meta

type meta struct {
	index   map[string]int // field -> 位下标
	names   []string       // field名
	bits    int            // 字段总数
	uint64n int            // 需要多少个 uint64
}

type GlobalBitSet[T GlobalModel] struct {
	set []uint64
}

// NewZero 返回一个全 0 的位图
func NewZero[T GlobalModel]() GlobalBitSet[T] {
	var zero T
	m := buildMeta(reflect.TypeOf(zero))
	return GlobalBitSet[T]{set: make([]uint64, m.uint64n)}
}

// NewAll 返回一个全 1 的位图
func NewAll[T GlobalModel]() GlobalBitSet[T] {
	var zero T
	m := buildMeta(reflect.TypeOf(zero))
	b := GlobalBitSet[T]{set: make([]uint64, m.uint64n)}
	for i := range b.set {
		b.set[i] = ^uint64(0)
	}
	// 高位可能多余，清零
	if extra := m.bits % 64; extra != 0 {
		b.set[len(b.set)-1] &= (1 << extra) - 1
	}
	return b
}

// meate 获取元数据
func (b *GlobalBitSet[T]) meta() *meta {
	var zero T
	return buildMeta(reflect.TypeOf(zero))
}

// Set 设置标记
func (b *GlobalBitSet[T]) Set(field string) *GlobalBitSet[T] {
	if idx, ok := b.meta().index[field]; !ok {
		panic(fmt.Sprintf("BitSet: field %q not found", field))
	} else {
		b.set[idx/64] |= 1 << (idx % 64)
	}
	return b
}

// Get 查标记
func (b *GlobalBitSet[T]) Get(field string) bool {
	if idx, ok := b.meta().index[field]; !ok {
		return false
	} else {
		return b.set[idx/64]&(1<<(idx%64)) != 0
	}
}

// Clear 清标记
func (b *GlobalBitSet[T]) Clear(field string) *GlobalBitSet[T] {
	if idx, ok := b.meta().index[field]; !ok {
		panic(fmt.Sprintf("BitSet: field %q not found", field))
	} else {
		b.set[idx/64] &^= 1 << (idx % 64)
	}
	return b
}

// Merge 合并另一个位图(或操作)
func (b *GlobalBitSet[T]) Merge(other GlobalBitSet[T]) *GlobalBitSet[T] {
	for i := range b.set {
		b.set[i] |= other.set[i]
	}
	return b
}

// ClearAll 全清 0
func (b *GlobalBitSet[T]) ClearAll() *GlobalBitSet[T] {
	for i := range b.set {
		b.set[i] = 0
	}
	return b
}

// SetAll 全置 1
func (b *GlobalBitSet[T]) SetAll() *GlobalBitSet[T] {
	for i := range b.set {
		b.set[i] = ^uint64(0)
	}

	if extra := b.meta().bits % 64; extra != 0 {
		b.set[len(b.set)-1] &= (1 << extra) - 1
	}
	return b
}

// IsSetAll 是否所有位都为 1
func (b *GlobalBitSet[T]) IsSetAll() bool {
	for i, v := range b.set {
		want := ^uint64(0)
		if i == len(b.set)-1 {
			if extra := b.meta().bits % 64; extra != 0 {
				want = (1 << extra) - 1
			}
		}
		if v != want {
			return false
		}
	}
	return true
}

// Fields 返回当前所有被置 1 的字段名，方便调试
func (b *GlobalBitSet[T]) Fields() []string {
	m := b.meta()
	var out []string
	for name, _ := range m.index {
		if b.Get(name) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// buildMeta 一次性反射，生成元数据
func buildMeta(t reflect.Type) *meta {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 缓存命中
	if m, ok := cache.Load(t); ok {
		return m.(*meta)
	}

	// 不是指针退出
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("BitSet: %v is not a struct", t))
	}

	n := t.NumField()
	names := make([]string, 0, n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		// 跳过没有导出的字段
		if !f.IsExported() {
			continue
		}
		// 手机所有的字段名字
		names = append(names, f.Name)
	}

	// 按字母序，保证顺序稳定
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Compare(a, b)
	})
	// 收集下标对应的字段状态
	index := make(map[string]int, len(names))
	for i, name := range names {
		index[name] = i
	}
	bits := len(names)
	uint64n := (bits + 63) / 64
	m := &meta{index: index, names: names, bits: bits, uint64n: uint64n}
	// 缓存bit meta
	cache.Store(t, m)
	return m
}
