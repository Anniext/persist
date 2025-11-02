// Package data 提供完全泛型化的持久化管理器
// 只需注入模型类型即可使用所有功能
package data

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"xorm.io/xorm"
)

// GenericManager 完全泛型化的管理器
// T: 数据模型类型
// 自动支持所有索引和操作，无需额外实现
type GenericManager[T any] struct {
	// 管理器状态
	managerState int32
	loadAll      int32

	// 数据库引擎
	engine *xorm.Engine

	// 主存储：使用反射提取的主键作为键
	storage *SyncMap[any, *T]

	// 索引映射：字段名 -> 索引
	singleIndexes map[string]*SyncMap[any, *T]                 // 单值索引
	multiIndexes  map[string]*SyncMap[any, *SyncMap[*T, bool]] // 多值索引

	// 对象池
	pool *sync.Pool

	// 同步控制
	syncChan chan *syncOp[T]
	exitChan chan bool
	mu       sync.RWMutex
}

// syncOp 同步操作
type syncOp[T any] struct {
	op   int8 // 1:insert 2:update 3:delete
	data *T
}

// NewGenericManager 创建泛型管理器
// 自动分析模型结构，创建索引
func NewGenericManager[T any](engine *xorm.Engine) *GenericManager[T] {
	m := &GenericManager[T]{
		engine:        engine,
		storage:       &SyncMap[any, *T]{},
		singleIndexes: make(map[string]*SyncMap[any, *T]),
		multiIndexes:  make(map[string]*SyncMap[any, *SyncMap[*T, bool]]),
		syncChan:      make(chan *syncOp[T], 1000),
		exitChan:      make(chan bool),
		pool:          &sync.Pool{New: func() interface{} { var t T; return &t }},
	}

	// 自动分析模型结构，创建索引
	m.analyzeModel()

	// 启动后台处理
	go m.processSync()

	return m
}

// analyzeModel 分析模型结构，自动创建索引
func (m *GenericManager[T]) analyzeModel() {
	var t T
	typ := reflect.TypeOf(t)

	// 遍历所有字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// 检查 xorm 标签中的 pk（主键）
		xormTag := field.Tag.Get("xorm")
		if contains(xormTag, "pk") {
			// 主键字段，已经用 storage 存储
			continue
		}

		// 检查 hash 标签
		hashTag := field.Tag.Get("hash")
		if hashTag != "" {
			// 解析 hash 标签
			// hash:"group=1;unique=1" -> 单值索引
			// hash:"group=3;unique=0" -> 多值索引
			if contains(hashTag, "unique=1") {
				// 创建单值索引
				m.singleIndexes[field.Name] = &SyncMap[any, *T]{}
			} else if contains(hashTag, "unique=0") {
				// 创建多值索引
				m.multiIndexes[field.Name] = &SyncMap[any, *SyncMap[*T, bool]]{}
			}
		}
	}
}

// Insert 插入数据
func (m *GenericManager[T]) Insert(obj *T) error {
	if obj == nil {
		return nil
	}

	// 提取主键
	pk := m.extractPrimaryKey(obj)

	// 存储到主存储
	m.storage.Store(pk, obj)

	// 更新所有索引
	m.updateIndexes(obj, true)

	// 发送同步操作
	m.syncChan <- &syncOp[T]{op: 1, data: obj}

	return nil
}

// Get 通过主键获取数据
func (m *GenericManager[T]) Get(pk any) (*T, bool) {
	return m.storage.Load(pk)
}

// GetByField 通过字段值获取数据（单值索引）
func (m *GenericManager[T]) GetByField(fieldName string, value any) (*T, bool) {
	m.mu.RLock()
	index, exists := m.singleIndexes[fieldName]
	m.mu.RUnlock()

	if !exists {
		return nil, false
	}

	return index.Load(value)
}

// GetAllByField 通过字段值获取所有数据（多值索引）
func (m *GenericManager[T]) GetAllByField(fieldName string, value any) []*T {
	m.mu.RLock()
	index, exists := m.multiIndexes[fieldName]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	set, ok := index.Load(value)
	if !ok {
		return nil
	}

	result := make([]*T, 0)
	set.Range(func(obj *T, _ bool) bool {
		result = append(result, obj)
		return true
	})

	return result
}

// Update 更新数据
func (m *GenericManager[T]) Update(obj *T) error {
	if obj == nil {
		return nil
	}

	pk := m.extractPrimaryKey(obj)

	// 获取旧数据
	old, exists := m.storage.Load(pk)
	if exists {
		// 从索引中删除旧数据
		m.updateIndexes(old, false)
	}

	// 存储新数据
	m.storage.Store(pk, obj)

	// 添加到索引
	m.updateIndexes(obj, true)

	// 发送同步操作
	m.syncChan <- &syncOp[T]{op: 2, data: obj}

	return nil
}

// Delete 删除数据
func (m *GenericManager[T]) Delete(obj *T) error {
	if obj == nil {
		return nil
	}

	pk := m.extractPrimaryKey(obj)

	// 从主存储删除
	m.storage.Delete(pk)

	// 从索引删除
	m.updateIndexes(obj, false)

	// 发送同步操作
	m.syncChan <- &syncOp[T]{op: 3, data: obj}

	return nil
}

// DeleteByPK 通过主键删除
func (m *GenericManager[T]) DeleteByPK(pk any) error {
	obj, ok := m.storage.Load(pk)
	if !ok {
		return nil
	}
	return m.Delete(obj)
}

// Range 遍历所有数据
func (m *GenericManager[T]) Range(f func(*T) bool) {
	m.storage.Range(func(_ any, obj *T) bool {
		return f(obj)
	})
}

// RangeByField 遍历指定字段值的所有数据
func (m *GenericManager[T]) RangeByField(fieldName string, value any, f func(*T) bool) {
	m.mu.RLock()
	index, exists := m.multiIndexes[fieldName]
	m.mu.RUnlock()

	if !exists {
		return
	}

	set, ok := index.Load(value)
	if !ok {
		return
	}

	set.Range(func(obj *T, _ bool) bool {
		return f(obj)
	})
}

// Count 统计数量
func (m *GenericManager[T]) Count() int {
	count := 0
	m.storage.Range(func(_ any, _ *T) bool {
		count++
		return true
	})
	return count
}

// CountByField 统计指定字段值的数量
func (m *GenericManager[T]) CountByField(fieldName string, value any) int {
	objs := m.GetAllByField(fieldName, value)
	return len(objs)
}

// Clear 清空所有数据
func (m *GenericManager[T]) Clear() {
	m.storage.Clear()

	m.mu.Lock()
	for _, index := range m.singleIndexes {
		index.Clear()
	}
	for _, index := range m.multiIndexes {
		index.Clear()
	}
	m.mu.Unlock()
}

// extractPrimaryKey 提取主键值
func (m *GenericManager[T]) extractPrimaryKey(obj *T) any {
	if obj == nil {
		return nil
	}

	val := reflect.ValueOf(obj).Elem()
	typ := val.Type()

	// 查找主键字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		xormTag := field.Tag.Get("xorm")

		if contains(xormTag, "pk") {
			// 返回主键值
			fieldVal := val.Field(i)
			return fieldVal.Interface()
		}
	}

	// 如果没有找到主键，返回对象指针
	return obj
}

// extractFieldValue 提取字段值
func (m *GenericManager[T]) extractFieldValue(obj *T, fieldName string) any {
	if obj == nil {
		return nil
	}

	val := reflect.ValueOf(obj).Elem()
	fieldVal := val.FieldByName(fieldName)

	if !fieldVal.IsValid() {
		return nil
	}

	return fieldVal.Interface()
}

// updateIndexes 更新索引
func (m *GenericManager[T]) updateIndexes(obj *T, add bool) {
	if obj == nil {
		return
	}

	val := reflect.ValueOf(obj).Elem()

	// 更新单值索引
	m.mu.RLock()
	for fieldName, index := range m.singleIndexes {
		fieldVal := val.FieldByName(fieldName)
		if fieldVal.IsValid() {
			key := fieldVal.Interface()
			if add {
				index.Store(key, obj)
			} else {
				index.Delete(key)
			}
		}
	}

	// 更新多值索引
	for fieldName, index := range m.multiIndexes {
		fieldVal := val.FieldByName(fieldName)
		if fieldVal.IsValid() {
			key := fieldVal.Interface()

			if add {
				// 添加到多值索引
				set, _ := index.LoadOrStore(key, &SyncMap[*T, bool]{})
				set.Store(obj, true)
			} else {
				// 从多值索引删除
				if set, ok := index.Load(key); ok {
					set.Delete(obj)
				}
			}
		}
	}
	m.mu.RUnlock()
}

// processSync 处理同步操作
func (m *GenericManager[T]) processSync() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]*syncOp[T], 0, 100)

	for {
		select {
		case op := <-m.syncChan:
			batch = append(batch, op)

			// 批量处理
			if len(batch) >= 100 {
				m.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				m.flushBatch(batch)
				batch = batch[:0]
			}

		case <-m.exitChan:
			// 处理剩余数据
			if len(batch) > 0 {
				m.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch 批量写入数据库
func (m *GenericManager[T]) flushBatch(batch []*syncOp[T]) {
	if m.engine == nil {
		return
	}

	session := m.engine.NewSession()
	defer session.Close()

	for _, op := range batch {
		switch op.op {
		case 1: // insert
			session.Insert(op.data)
		case 2: // update
			session.Update(op.data)
		case 3: // delete
			session.Delete(op.data)
		}
	}
}

// Run 启动管理器
func (m *GenericManager[T]) Run() error {
	atomic.StoreInt32(&m.managerState, 1) // 1 = Normal
	return nil
}

// Exit 退出管理器
func (m *GenericManager[T]) Exit() error {
	m.exitChan <- true
	return nil
}

// Dead 检查管理器是否出错
func (m *GenericManager[T]) Dead() bool {
	return atomic.LoadInt32(&m.managerState) != 1 // 1 = Normal
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
