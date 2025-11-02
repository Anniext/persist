# GlobalManager 泛型设计说明

## 概述

`GlobalManager` 是一个支持泛型的全局对象管理器，提供了灵活的复合主键支持。通过泛型设计，可以适配不同的模型和复合主键类型。

## 核心接口

### 1. GlobalKeyTypeHashCompoundPrimaryId

复合主键接口，所有复合主键类型必须实现此接口：

```go
type GlobalKeyTypeHashCompoundPrimaryId interface {
    comparable  // 必须可比较，以便作为 map 的 key
    isCompoundPrimaryId()
}
```

### 2. GlobalCompoundKeyProvider[K]

模型可选实现的接口，用于提供复合主键：

```go
type GlobalCompoundKeyProvider[K GlobalKeyTypeHashCompoundPrimaryId] interface {
    GetCompoundKey() K
}
```

## 使用方式

### 方式一：接口实现（推荐）

**优点：**

- 类型安全
- 代码清晰
- 易于维护
- 支持 IDE 自动补全

**步骤：**

1. 定义复合主键类型：

```go
type MenusGlobalCompoundKey struct {
    AuthId int64
    Type   string
}

func (MenusGlobalCompoundKey) isCompoundPrimaryId() {}
```

2. 让模型实现 `GlobalCompoundKeyProvider` 接口：

```go
func (m *MenusGlobal) GetCompoundKey() MenusGlobalCompoundKey {
    return MenusGlobalCompoundKey{
        AuthId: m.AuthId,
        Type:   m.Type,
    }
}
```

3. 创建 Manager：

```go
manager := NewGlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey](engine, nil)
```

### 方式二：提取函数

**优点：**

- 无需修改模型代码
- 灵活性高
- 适合第三方模型

**步骤：**

1. 定义复合主键类型（同方式一）

2. 定义提取函数：

```go
func extractKey(m *model.MenusGlobal) MenusGlobalCompoundKey {
    return MenusGlobalCompoundKey{
        AuthId: m.AuthId,
        Type:   m.Type,
    }
}
```

3. 创建 Manager 时传入提取函数：

```go
manager := NewGlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey](
    engine,
    extractKey,
)
```

## 复合主键类型示例

### 双字段主键

```go
type UserRoleKey struct {
    UserId int64
    RoleId int64
}

func (UserRoleKey) isCompoundPrimaryId() {}
```

### 三字段主键

```go
type OrderItemKey struct {
    OrderId  int64
    ItemId   int64
    Sequence int32
}

func (OrderItemKey) isCompoundPrimaryId() {}
```

### 字符串主键

```go
type TenantResourceKey struct {
    TenantId   string
    ResourceId string
}

func (TenantResourceKey) isCompoundPrimaryId() {}
```

### 混合类型主键

```go
type MixedKey struct {
    OrgId    int64
    Category string
    Status   int32
}

func (MixedKey) isCompoundPrimaryId() {}
```

## 工作原理

### 主键索引

使用 `PrimarySyncMap[T]` 存储主键到对象的映射：

```
主键 (int64) -> 对象 (T)
```

### 复合主键索引

使用 `sync.Map` 存储复合主键到对象集合的映射：

```
复合主键 (K) -> SetSyncMap[T] -> 对象集合
```

### 添加对象流程

```go
func (m *GlobalManager[T, K]) addGlobal(cls T) (data T, ok bool) {
    // 1. 添加到主键索引
    actual, loaded := m.hasPrimaryId.LoadOrStore(
        PrimaryKeyTypeHashPrimaryId(cls.isPrimaryId()),
        cls,
    )

    if !loaded {
        // 2. 获取复合主键
        if compoundKey, hasKey := m.getCompoundKey(cls); hasKey {
            // 3. 添加到复合主键索引
            setInterface, _ := m.hasCompoundPrimaryId.LoadOrStore(
                compoundKey,
                &SetSyncMap[T]{},
            )
            if set, ok := setInterface.(*SetSyncMap[T]); ok {
                set.Store(cls, true)
            }
        }
    }

    return actual, !loaded
}
```

### 复合主键获取流程

```go
func (m *GlobalManager[T, K]) getCompoundKey(cls T) (key K, ok bool) {
    // 优先级1: 检查是否实现了接口
    if provider, ok := any(cls).(GlobalCompoundKeyProvider[K]); ok {
        return provider.GetCompoundKey(), true
    }

    // 优先级2: 使用提取函数
    if m.compoundKeyExtractor != nil {
        return m.compoundKeyExtractor(cls), true
    }

    // 没有提供复合主键
    var zero K
    return zero, false
}
```

## 设计优势

1. **类型安全**：编译时检查类型正确性
2. **灵活性**：支持任意复合主键结构
3. **可扩展**：易于添加新的索引类型
4. **性能**：使用并发安全的 Map 实现
5. **解耦**：复合主键定义与模型分离

## 注意事项

1. 复合主键类型必须是 `comparable`，即可以用 `==` 比较
2. 复合主键字段应该是不可变的（immutable）
3. 建议使用接口实现方式，代码更清晰
4. 如果不需要复合主键索引，可以定义一个空的复合主键类型

## 空复合主键示例

如果某个模型不需要复合主键索引：

```go
// 定义空的复合主键类型
type EmptyCompoundKey struct{}

func (EmptyCompoundKey) isCompoundPrimaryId() {}

// 创建 Manager 时不提供提取函数
manager := NewGlobalManager[*model.SimpleModel, EmptyCompoundKey](engine, nil)

// 模型不需要实现 GetCompoundKey() 方法
```

## 迁移指南

### 从旧版本迁移

旧代码：

```go
manager := NewGlobalManager[*model.MenusGlobal](engine)
```

新代码：

```go
// 1. 定义复合主键类型
type MenusGlobalCompoundKey struct {
    AuthId int64
    Type   string
}

func (MenusGlobalCompoundKey) isCompoundPrimaryId() {}

// 2. 更新 Manager 创建
manager := NewGlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey](
    engine,
    func(m *model.MenusGlobal) MenusGlobalCompoundKey {
        return MenusGlobalCompoundKey{
            AuthId: m.AuthId,
            Type:   m.Type,
        }
    },
)
```

## 常见问题

### Q: 为什么需要两个泛型参数？

A: `T` 是模型类型，`K` 是复合主键类型。这样设计可以让复合主键的结构完全由外部定义，不受模型限制。

### Q: 可以不使用复合主键吗？

A: 可以。定义一个空的复合主键类型，并且不提供提取函数即可。

### Q: 接口方式和函数方式哪个更好？

A: 推荐使用接口方式，因为：

- 类型安全性更好
- IDE 支持更完善
- 代码可读性更高
- 便于单元测试

### Q: 复合主键可以包含指针字段吗？

A: 不建议。复合主键应该是值类型，且字段应该是不可变的基本类型。
