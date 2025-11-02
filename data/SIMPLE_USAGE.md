# 完全泛型化使用指南

## 核心理念

**只需注入模型类型，无需任何额外实现！**

```go
// 就这么简单！
manager := NewGenericManager[model.MenusGlobal](engine)
```

管理器会自动：

- ✅ 分析模型结构
- ✅ 识别主键（xorm:"pk"）
- ✅ 创建索引（hash 标签）
- ✅ 提供所有 CRUD 操作
- ✅ 支持并发安全
- ✅ 自动持久化到数据库

## 快速开始

### 1. 创建管理器

```go
import "github.com/spelens-gud/persist/data"

// 为任何模型创建管理器
menuManager := data.NewGenericManager[model.MenusGlobal](engine)
userManager := data.NewGenericManager[model.User](engine)
roleManager := data.NewGenericManager[model.Role](engine)
```

### 2. 基本操作

```go
// 插入
menu := &model.MenusGlobal{
    AuthId: 1,
    Name:   "Dashboard",
    Type:   "menu",
}
manager.Insert(menu)

// 查询（通过主键）
if menu, ok := manager.Get(int64(1)); ok {
    fmt.Println(menu.Name)
}

// 更新
menu.Name = "Dashboard Updated"
manager.Update(menu)

// 删除
manager.Delete(menu)

// 或通过主键删除
manager.DeleteByPK(int64(1))
```

### 3. 索引查询

管理器会自动识别模型中的 hash 标签并创建索引：

```go
// 模型定义（已有）
type MenusGlobal struct {
    AuthId int64  `xorm:"pk" hash:"group=1;unique=1"`  // 自动创建单值索引
    Type   string `hash:"group=3;unique=0"`             // 自动创建多值索引
    Name   string
}

// 使用索引查询
// 单值索引：一个键对应一个值
if menu, ok := manager.GetByField("AuthId", int64(1)); ok {
    fmt.Println(menu.Name)
}

// 多值索引：一个键对应多个值
menus := manager.GetAllByField("Type", "menu")
for _, m := range menus {
    fmt.Println(m.Name)
}
```

### 4. 遍历操作

```go
// 遍历所有数据
manager.Range(func(menu *model.MenusGlobal) bool {
    fmt.Println(menu.Name)
    return true  // 返回 false 停止遍历
})

// 遍历指定字段值的数据
manager.RangeByField("Type", "menu", func(menu *model.MenusGlobal) bool {
    fmt.Println(menu.Name)
    return true
})
```

### 5. 统计操作

```go
// 统计总数
total := manager.Count()

// 统计指定字段值的数量
menuCount := manager.CountByField("Type", "menu")
buttonCount := manager.CountByField("Type", "button")
```

### 6. 清空数据

```go
// 清空所有数据（包括索引）
manager.Clear()
```

## 完整示例

```go
package main

import (
    "fmt"
    "github.com/spelens-gud/persist/data"
    "github.com/spelens-gud/persist/model"
)

func main() {
    // 1. 创建管理器（只需这一行！）
    manager := data.NewGenericManager[model.MenusGlobal](nil)

    // 2. 插入数据
    menus := []*model.MenusGlobal{
        {AuthId: 1, Name: "Dashboard", Type: "menu"},
        {AuthId: 2, Name: "Users", Type: "menu"},
        {AuthId: 3, Name: "Settings", Type: "menu"},
        {AuthId: 4, Name: "Add User", Type: "button"},
        {AuthId: 5, Name: "Delete User", Type: "button"},
    }

    for _, menu := range menus {
        manager.Insert(menu)
    }

    // 3. 查询单个
    if menu, ok := manager.Get(int64(1)); ok {
        fmt.Printf("Found: %s\n", menu.Name)
    }

    // 4. 查询列表
    menuList := manager.GetAllByField("Type", "menu")
    fmt.Printf("Found %d menus:\n", len(menuList))
    for _, m := range menuList {
        fmt.Printf("  - %s\n", m.Name)
    }

    // 5. 遍历
    fmt.Println("\nAll items:")
    manager.Range(func(m *model.MenusGlobal) bool {
        fmt.Printf("  %d: %s (%s)\n", m.AuthId, m.Name, m.Type)
        return true
    })

    // 6. 统计
    fmt.Printf("\nTotal: %d\n", manager.Count())
    fmt.Printf("Menus: %d\n", manager.CountByField("Type", "menu"))
    fmt.Printf("Buttons: %d\n", manager.CountByField("Type", "button"))

    // 7. 更新
    menus[0].Name = "Dashboard Updated"
    manager.Update(menus[0])

    // 8. 删除
    manager.Delete(menus[0])
    fmt.Printf("\nAfter delete: %d items\n", manager.Count())
}
```

输出：

```
Found: Dashboard
Found 3 menus:
  - Dashboard
  - Users
  - Settings

All items:
  1: Dashboard (menu)
  2: Users (menu)
  3: Settings (menu)
  4: Add User (button)
  5: Delete User (button)

Total: 5
Menus: 3
Buttons: 2

After delete: 4 items
```

## 支持的模型标签

### xorm 标签

```go
type Model struct {
    ID int64 `xorm:"pk autoincr"`  // pk: 主键标识
}
```

### hash 标签

```go
type Model struct {
    // unique=1: 单值索引（一对一）
    AuthId int64 `hash:"group=1;unique=1"`

    // unique=0: 多值索引（一对多）
    Type string `hash:"group=3;unique=0"`
}
```

## 并发安全

所有操作都是并发安全的，可以在多个 goroutine 中使用：

```go
// 并发插入
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        menu := &model.MenusGlobal{
            AuthId: int64(id),
            Name:   fmt.Sprintf("Menu %d", id),
            Type:   "menu",
        }
        manager.Insert(menu)
    }(i)
}
wg.Wait()

// 并发查询
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        manager.Get(int64(id))
    }(i)
}
wg.Wait()
```

## 自动持久化

如果传入了数据库引擎，管理器会自动批量持久化数据：

```go
// 创建时传入引擎
manager := data.NewGenericManager[model.MenusGlobal](engine)

// 所有操作会自动持久化到数据库
manager.Insert(menu)   // 自动写入数据库
manager.Update(menu)   // 自动更新数据库
manager.Delete(menu)   // 自动从数据库删除
```

批量策略：

- 每 100 条记录批量写入
- 或每 100ms 批量写入
- 自动优化性能

## 对比原有实现

### 原有方式（需要 4 个文件，~3100 行代码）

```go
// 需要生成的文件：
// - 001_menusglobal_persist.go    (~1500 行)
// - 001_menusglobal_set.go        (~600 行)
// - 001_menusglobalhashauthid.go  (~500 行)
// - 001_menusglobalhashauthidtype.go (~500 行)

// 使用
manager := NewMenusGlobalManager(engine)
menu := NewMenusGlobal(1)
manager.GetMenusGlobalByAuthId(1)
manager.GetMenusGlobalByAuthIdType(1, "menu")
```

### 新方式（0 个额外文件，直接使用）

```go
// 只需要一行！
manager := data.NewGenericManager[model.MenusGlobal](engine)

// 统一的接口
menu, _ := manager.Get(1)
menu, _ := manager.GetByField("AuthId", 1)
menus := manager.GetAllByField("Type", "menu")
```

## 优势总结

| 特性     | 原有方式      | 泛型方式    |
| -------- | ------------- | ----------- |
| 代码量   | 3100 行/模型  | 0 行/模型   |
| 文件数   | 4 个/模型     | 0 个/模型   |
| 类型安全 | ✅            | ✅          |
| 并发安全 | ✅            | ✅          |
| 自动索引 | ❌ 需手动配置 | ✅ 自动识别 |
| 学习成本 | 高            | 低          |
| 维护成本 | 高            | 低          |
| 扩展性   | 低            | 高          |

## 常见问题

### Q: 如何为新模型创建管理器？

A: 只需一行代码：

```go
manager := data.NewGenericManager[YourModel](engine)
```

### Q: 如何添加索引？

A: 在模型定义中添加 hash 标签：

```go
type YourModel struct {
    ID   int64  `xorm:"pk" hash:"group=1;unique=1"`
    Type string `hash:"group=2;unique=0"`
}
```

### Q: 性能如何？

A: 与原有实现相当或更好：

- 使用相同的底层数据结构
- 泛型零成本抽象
- 批量持久化优化

### Q: 可以与原有代码共存吗？

A: 可以！新旧代码可以并存，逐步迁移。

## 下一步

- ✅ 开始使用：`manager := data.NewGenericManager[YourModel](engine)`
- 📖 查看测试：`data/generic_manager_test.go`
- 🚀 迁移现有代码：逐步替换旧的管理器

---

**就是这么简单！只需注入类型，立即使用！**
