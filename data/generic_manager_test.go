package data

import (
	"testing"

	"github.com/spelens-gud/persist/model"
)

// TestGenericManagerBasic 测试基本功能
func TestGenericManagerBasic(t *testing.T) {
	// 创建管理器 - 只需要传入类型！
	manager := NewGenericManager[model.MenusGlobal](nil)

	// 插入数据
	menu1 := &model.MenusGlobal{
		AuthId: 1,
		Name:   "Dashboard",
		Type:   "menu",
	}
	menu2 := &model.MenusGlobal{
		AuthId: 2,
		Name:   "Users",
		Type:   "menu",
	}
	menu3 := &model.MenusGlobal{
		AuthId: 3,
		Name:   "Add User",
		Type:   "button",
	}

	manager.Insert(menu1)
	manager.Insert(menu2)
	manager.Insert(menu3)

	// 通过主键查询
	if menu, ok := manager.Get(int64(1)); !ok || menu.Name != "Dashboard" {
		t.Error("Failed to get by primary key")
	}

	// 通过字段查询（单值索引）
	if menu, ok := manager.GetByField("AuthId", int64(1)); !ok || menu.Name != "Dashboard" {
		t.Error("Failed to get by field")
	}

	// 通过字段查询（多值索引）
	menus := manager.GetAllByField("Type", "menu")
	if len(menus) != 2 {
		t.Errorf("Expected 2 menus, got %d", len(menus))
	}

	// 统计
	if count := manager.Count(); count != 3 {
		t.Errorf("Expected 3 items, got %d", count)
	}

	if count := manager.CountByField("Type", "menu"); count != 2 {
		t.Errorf("Expected 2 menus, got %d", count)
	}

	// 遍历
	count := 0
	manager.Range(func(m *model.MenusGlobal) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("Expected 3 items in range, got %d", count)
	}

	// 按字段遍历
	count = 0
	manager.RangeByField("Type", "menu", func(m *model.MenusGlobal) bool {
		count++
		return true
	})
	if count != 2 {
		t.Errorf("Expected 2 menus in range, got %d", count)
	}

	// 更新
	menu1.Name = "Dashboard Updated"
	manager.Update(menu1)
	if menu, ok := manager.Get(int64(1)); !ok || menu.Name != "Dashboard Updated" {
		t.Error("Failed to update")
	}

	// 删除
	manager.Delete(menu1)
	if _, ok := manager.Get(int64(1)); ok {
		t.Error("Failed to delete")
	}

	// 通过主键删除
	manager.DeleteByPK(int64(2))
	if _, ok := manager.Get(int64(2)); ok {
		t.Error("Failed to delete by PK")
	}

	// 清空
	manager.Clear()
	if count := manager.Count(); count != 0 {
		t.Errorf("Expected 0 items after clear, got %d", count)
	}
}

// TestGenericManagerConcurrent 测试并发安全
func TestGenericManagerConcurrent(t *testing.T) {
	manager := NewGenericManager[model.MenusGlobal](nil)

	// 并发插入
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			menu := &model.MenusGlobal{
				AuthId: int64(id),
				Name:   "Menu",
				Type:   "menu",
			}
			manager.Insert(menu)
			done <- true
		}(i)
	}

	// 等待完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证
	if count := manager.Count(); count != 10 {
		t.Errorf("Expected 10 items, got %d", count)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		go func(id int) {
			manager.Get(int64(id))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestGenericManagerMultipleTypes 测试多种类型
func TestGenericManagerMultipleTypes(t *testing.T) {
	// MenusGlobal 管理器
	menuManager := NewGenericManager[model.MenusGlobal](nil)
	menu := &model.MenusGlobal{AuthId: 1, Name: "Test"}
	menuManager.Insert(menu)

	if m, ok := menuManager.Get(int64(1)); !ok || m.Name != "Test" {
		t.Error("MenusGlobal manager failed")
	}

	// 可以为其他模型创建管理器
	// userManager := NewGenericManager[model.User](nil)
	// roleManager := NewGenericManager[model.Role](nil)
	// 等等...
}

// BenchmarkGenericManagerInsert 基准测试：插入
func BenchmarkGenericManagerInsert(b *testing.B) {
	manager := NewGenericManager[model.MenusGlobal](nil)
	menu := &model.MenusGlobal{AuthId: 1, Name: "Test", Type: "menu"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		menu.AuthId = int64(i)
		manager.Insert(menu)
	}
}

// BenchmarkGenericManagerGet 基准测试：查询
func BenchmarkGenericManagerGet(b *testing.B) {
	manager := NewGenericManager[model.MenusGlobal](nil)

	// 准备数据
	for i := 0; i < 1000; i++ {
		menu := &model.MenusGlobal{AuthId: int64(i), Name: "Test", Type: "menu"}
		manager.Insert(menu)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Get(int64(i % 1000))
	}
}

// BenchmarkGenericManagerGetByField 基准测试：字段查询
func BenchmarkGenericManagerGetByField(b *testing.B) {
	manager := NewGenericManager[model.MenusGlobal](nil)

	// 准备数据
	for i := 0; i < 1000; i++ {
		menu := &model.MenusGlobal{AuthId: int64(i), Name: "Test", Type: "menu"}
		manager.Insert(menu)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetAllByField("Type", "menu")
	}
}

// ExampleGenericManager 使用示例
func ExampleGenericManager() {
	// 1. 创建管理器 - 只需要传入模型类型！
	manager := NewGenericManager[model.MenusGlobal](nil)

	// 2. 插入数据
	menu := &model.MenusGlobal{
		AuthId: 1,
		Name:   "Dashboard",
		Type:   "menu",
	}
	manager.Insert(menu)

	// 3. 通过主键查询
	if m, ok := manager.Get(int64(1)); ok {
		println("Found:", m.Name)
	}

	// 4. 通过字段查询（自动识别索引）
	if m, ok := manager.GetByField("AuthId", int64(1)); ok {
		println("Found by field:", m.Name)
	}

	// 5. 查询多个（多值索引）
	menus := manager.GetAllByField("Type", "menu")
	println("Found", len(menus), "menus")

	// 6. 遍历
	manager.Range(func(m *model.MenusGlobal) bool {
		println(m.Name)
		return true
	})

	// 7. 按字段遍历
	manager.RangeByField("Type", "menu", func(m *model.MenusGlobal) bool {
		println(m.Name)
		return true
	})

	// 8. 更新
	menu.Name = "Dashboard Updated"
	manager.Update(menu)

	// 9. 删除
	manager.Delete(menu)

	// 10. 统计
	count := manager.Count()
	println("Total:", count)
}
