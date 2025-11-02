package persist

import "github.com/spelens-gud/persist/model"

// 示例：如何使用泛型 GlobalManager
//
// GlobalManager 支持两种方式提供复合主键：
// 1. 让模型实现 GlobalCompoundKeyProvider[K] 接口（推荐）
// 2. 在创建 Manager 时提供 keyExtractor 函数

// ============================================================================
// 方式1: 通过接口实现（推荐）
// ============================================================================

// 步骤1: 定义复合主键类型
type MenusGlobalCompoundKey struct {
	AuthId int64
	Type   string
}

// 步骤2: 实现 GlobalKeyTypeHashCompoundPrimaryId 接口
func (MenusGlobalCompoundKey) isCompoundPrimaryId() {}

// 步骤3: 让模型实现 GlobalCompoundKeyProvider 接口
// 在 model/menus.go 中添加：
/*
func (m *MenusGlobal) GetCompoundKey() MenusGlobalCompoundKey {
	return MenusGlobalCompoundKey{
		AuthId: m.AuthId,
		Type:   m.Type,
	}
}
*/

// 步骤4: 创建 Manager（keyExtractor 参数传 nil）
func ExampleUsageWithInterface() {
	// engine := getDBEngine()
	// manager := NewGlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey](engine, nil)
	//
	// // 添加对象时会自动调用 GetCompoundKey() 方法
	// menu := &model.MenusGlobal{AuthId: 1, Type: "menu"}
	// manager.addGlobal(menu)
}

// ============================================================================
// 方式2: 通过提取函数
// ============================================================================

// 步骤1: 定义复合主键类型（同上）
// 步骤2: 实现接口（同上）

// 步骤3: 定义提取函数
func menusGlobalKeyExtractor(m *model.MenusGlobal) MenusGlobalCompoundKey {
	return MenusGlobalCompoundKey{
		AuthId: m.AuthId,
		Type:   m.Type,
	}
}

// 步骤4: 创建 Manager 时传入提取函数
func ExampleUsageWithExtractor() {
	// engine := getDBEngine()
	// manager := NewGlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey](
	// 	engine,
	// 	menusGlobalKeyExtractor,
	// )
	//
	// // 添加对象时会自动调用提取函数
	// menu := &model.MenusGlobal{AuthId: 1, Type: "menu"}
	// manager.addGlobal(menu)
}

// ============================================================================
// 其他复合主键示例
// ============================================================================

// 双字段复合主键：用户角色关联
type UserRoleCompoundKey struct {
	UserId int64
	RoleId int64
}

func (UserRoleCompoundKey) isCompoundPrimaryId() {}

// 三字段复合主键：订单商品
type OrderItemCompoundKey struct {
	OrderId  int64
	ItemId   int64
	Sequence int32
}

func (OrderItemCompoundKey) isCompoundPrimaryId() {}

// 字符串类型复合主键
type TenantResourceKey struct {
	TenantId   string
	ResourceId string
}

func (TenantResourceKey) isCompoundPrimaryId() {}

// 混合类型复合主键
type MixedCompoundKey struct {
	OrgId    int64
	Category string
	Status   int32
}

func (MixedCompoundKey) isCompoundPrimaryId() {}

// ============================================================================
// 完整使用示例
// ============================================================================

/*
// 1. 在 model/menus.go 中定义模型
type MenusGlobal struct {
	AuthId   int64  `xorm:"pk"`
	ParentId int64
	Type     string
	Name     string
	// ... 其他字段
}

func (m *MenusGlobal) isPrimaryId() int64 { return m.AuthId }
func (m *MenusGlobal) isModel() {}

// 实现复合主键接口
func (m *MenusGlobal) GetCompoundKey() MenusGlobalCompoundKey {
	return MenusGlobalCompoundKey{
		AuthId: m.AuthId,
		Type:   m.Type,
	}
}

// 2. 在 data/ 中创建 Manager
var GMenusGlobalManager *GlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey]

func InitMenusGlobalManager(engine *xorm.Engine) {
	GMenusGlobalManager = NewGlobalManager[*model.MenusGlobal, MenusGlobalCompoundKey](
		engine,
		nil, // 使用接口方法，所以传 nil
	)
}

// 3. 使用 Manager
func CreateMenu(menu *model.MenusGlobal) error {
	actual, ok := GMenusGlobalManager.addGlobal(menu)
	if !ok {
		return errors.New("菜单已存在")
	}
	// ... 其他逻辑
	return nil
}

func GetMenuByPrimaryId(authId int64) *model.MenusGlobal {
	return GMenusGlobalManager.GetGlobalByPrimaryId(authId)
}
*/
