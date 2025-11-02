package data

import (
	"testing"

	"github.com/spelens-gud/persist/model"
)

// TestGenericSyncMap 测试泛型SyncMap
func TestGenericSyncMap(t *testing.T) {
	// 创建一个 int -> string 的并发安全Map
	m := &SyncMap[int, string]{}

	// 存储
	m.Store(1, "one")
	m.Store(2, "two")
	m.Store(3, "three")

	// 加载
	if val, ok := m.Load(1); !ok || val != "one" {
		t.Errorf("Expected 'one', got '%s'", val)
	}

	// LoadOrStore
	if val, loaded := m.LoadOrStore(4, "four"); loaded {
		t.Error("Expected not loaded")
	} else if val != "four" {
		t.Errorf("Expected 'four', got '%s'", val)
	}

	// 删除
	m.Delete(2)
	if _, ok := m.Load(2); ok {
		t.Error("Expected key 2 to be deleted")
	}

	// Range
	count := 0
	m.Range(func(key int, value string) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("Expected 3 items, got %d", count)
	}
}

// TestHashIndex 测试单值哈希索引
func TestHashIndex(t *testing.T) {
	// 创建索引
	index := NewHashIndex[int64, *model.MenusGlobal]()

	// 创建测试数据
	menu1 := &model.MenusGlobal{AuthId: 1, Name: "Menu1"}
	menu2 := &model.MenusGlobal{AuthId: 2, Name: "Menu2"}

	// 设置
	index.Set(1, menu1)
	index.Set(2, menu2)

	// 获取
	if val, ok := index.Get(1); !ok || val.Name != "Menu1" {
		t.Error("Failed to get menu1")
	}

	// 检查存在
	if !index.Has(1) {
		t.Error("Expected key 1 to exist")
	}

	// 删除
	index.Remove(1)
	if index.Has(1) {
		t.Error("Expected key 1 to be removed")
	}
}

// TestMultiHashIndex 测试多值哈希索引
func TestMultiHashIndex(t *testing.T) {
	// 创建索引
	index := NewMultiHashIndex[string, *model.MenusGlobal]()

	// 创建测试数据
	menu1 := &model.MenusGlobal{AuthId: 1, Name: "Menu1", Type: "menu"}
	menu2 := &model.MenusGlobal{AuthId: 2, Name: "Menu2", Type: "menu"}
	menu3 := &model.MenusGlobal{AuthId: 3, Name: "Menu3", Type: "button"}

	// 添加
	index.Add("menu", menu1)
	index.Add("menu", menu2)
	index.Add("button", menu3)

	// 获取所有
	menus := index.GetAll("menu")
	if len(menus) != 2 {
		t.Errorf("Expected 2 menus, got %d", len(menus))
	}

	// 检查
	if !index.Has("menu", menu1) {
		t.Error("Expected menu1 to exist")
	}

	// 删除单个
	index.Remove("menu", menu1)
	menus = index.GetAll("menu")
	if len(menus) != 1 {
		t.Errorf("Expected 1 menu after removal, got %d", len(menus))
	}

	// 删除所有
	index.RemoveAll("menu")
	menus = index.GetAll("menu")
	if len(menus) != 0 {
		t.Errorf("Expected 0 menus after RemoveAll, got %d", len(menus))
	}

	// 遍历
	count := 0
	index.RangeValues("button", func(value *model.MenusGlobal) bool {
		count++
		return true
	})
	if count != 1 {
		t.Errorf("Expected 1 button, got %d", count)
	}
}

// TestMenusGlobalBitSet 测试BitSet
func TestMenusGlobalBitSet(t *testing.T) {
	var bs MenusGlobalBitSet

	// 设置位
	bs.Set(EMenusGlobalFieldIndexAuthId)
	bs.Set(EMenusGlobalFieldIndexName)
	bs.Set(EMenusGlobalFieldIndexType)

	// 检查位
	if !bs.Get(EMenusGlobalFieldIndexAuthId) {
		t.Error("Expected AuthId bit to be set")
	}
	if !bs.Get(EMenusGlobalFieldIndexName) {
		t.Error("Expected Name bit to be set")
	}
	if bs.Get(EMenusGlobalFieldIndexParentId) {
		t.Error("Expected ParentId bit to be unset")
	}

	// 清除位
	bs.Clear(EMenusGlobalFieldIndexName)
	if bs.Get(EMenusGlobalFieldIndexName) {
		t.Error("Expected Name bit to be cleared")
	}

	// 合并
	var bs2 MenusGlobalBitSet
	bs2.Set(EMenusGlobalFieldIndexParentId)
	bs.Merge(bs2)
	if !bs.Get(EMenusGlobalFieldIndexParentId) {
		t.Error("Expected ParentId bit to be set after merge")
	}

	// 设置所有
	bs.SetAll()
	if !bs.IsSetAll() {
		t.Error("Expected all bits to be set")
	}

	// 清除所有
	bs.ClearAll()
	for i := MenusGlobalFieldIndex(0); i < EMenusGlobalFiledIndexLength; i++ {
		if bs.Get(i) {
			t.Errorf("Expected bit %d to be cleared", i)
		}
	}
}

// BenchmarkSyncMapStore 基准测试：存储
func BenchmarkSyncMapStore(b *testing.B) {
	m := &SyncMap[int, string]{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(i, "value")
	}
}

// BenchmarkSyncMapLoad 基准测试：加载
func BenchmarkSyncMapLoad(b *testing.B) {
	m := &SyncMap[int, string]{}
	for i := 0; i < 1000; i++ {
		m.Store(i, "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Load(i % 1000)
	}
}

// BenchmarkHashIndexSet 基准测试：哈希索引设置
func BenchmarkHashIndexSet(b *testing.B) {
	index := NewHashIndex[int64, *model.MenusGlobal]()
	menu := &model.MenusGlobal{AuthId: 1, Name: "Menu"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Set(int64(i), menu)
	}
}

// BenchmarkHashIndexGet 基准测试：哈希索引获取
func BenchmarkHashIndexGet(b *testing.B) {
	index := NewHashIndex[int64, *model.MenusGlobal]()
	menu := &model.MenusGlobal{AuthId: 1, Name: "Menu"}
	for i := 0; i < 1000; i++ {
		index.Set(int64(i), menu)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Get(int64(i % 1000))
	}
}

// BenchmarkMultiHashIndexAdd 基准测试：多值索引添加
func BenchmarkMultiHashIndexAdd(b *testing.B) {
	index := NewMultiHashIndex[string, *model.MenusGlobal]()
	menu := &model.MenusGlobal{AuthId: 1, Name: "Menu"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Add("menu", menu)
	}
}

// ExampleSyncMap 示例：使用泛型SyncMap
func ExampleSyncMap() {
	// 创建一个并发安全的 string -> int Map
	m := &SyncMap[string, int]{}

	// 存储键值对
	m.Store("one", 1)
	m.Store("two", 2)
	m.Store("three", 3)

	// 加载值
	if val, ok := m.Load("one"); ok {
		println("one =", val)
	}

	// 遍历所有键值对
	m.Range(func(key string, value int) bool {
		println(key, "=", value)
		return true // 继续遍历
	})

	// 删除键
	m.Delete("two")
}

// ExampleHashIndex 示例：使用单值哈希索引
func ExampleHashIndex() {
	// 创建索引：用户ID -> 用户对象
	type User struct {
		ID   int64
		Name string
	}

	index := NewHashIndex[int64, *User]()

	// 添加用户
	user1 := &User{ID: 1, Name: "Alice"}
	user2 := &User{ID: 2, Name: "Bob"}
	index.Set(1, user1)
	index.Set(2, user2)

	// 查询用户
	if user, ok := index.Get(1); ok {
		println("Found user:", user.Name)
	}

	// 删除用户
	index.Remove(1)
}

// ExampleMultiHashIndex 示例：使用多值哈希索引
func ExampleMultiHashIndex() {
	// 创建索引：部门 -> 员工列表
	type Employee struct {
		ID         int64
		Name       string
		Department string
	}

	index := NewMultiHashIndex[string, *Employee]()

	// 添加员工
	emp1 := &Employee{ID: 1, Name: "Alice", Department: "Engineering"}
	emp2 := &Employee{ID: 2, Name: "Bob", Department: "Engineering"}
	emp3 := &Employee{ID: 3, Name: "Charlie", Department: "Sales"}

	index.Add("Engineering", emp1)
	index.Add("Engineering", emp2)
	index.Add("Sales", emp3)

	// 获取部门所有员工
	engineers := index.GetAll("Engineering")
	println("Engineering has", len(engineers), "employees")

	// 遍历部门员工
	index.RangeValues("Engineering", func(emp *Employee) bool {
		println("Employee:", emp.Name)
		return true
	})

	// 删除单个员工
	index.Remove("Engineering", emp1)

	// 删除整个部门
	index.RemoveAll("Sales")
}
