package model

const (
	ViewStatue = 1
	HideStatue = 2
)

type MenusGlobal struct {
	AuthId             int64  `xorm:"pk" hash:"group=1;unique=1" hash:"group=3;unique=0"` // 权限id
	ParentId           int64  `xorm:""`                                                   // 父菜单ID
	TreePath           string `xorm:""`                                                   // 父节点ID路径
	Name               string `xorm:""`                                                   // 菜单名称
	Type               string `xorm:"" hash:"group=3;unique=0"`                           // 菜单类型
	RouteName          string `xorm:""`                                                   // 路由名称（Vue Router 中用于命名路由）
	Path               string `xorm:""`                                                   // 路由路径（Vue Router 中定义的 URL 路径）
	Component          string `xorm:""`                                                   // 组件路径（组件页面完整路径，相对于 src/views/，缺省后缀 .vue）
	Perm               string `xorm:""`                                                   // [按钮]权限标识
	Status             int64  `xorm:""`                                                   // 显示状态（1-显示 2-隐藏）
	AffixTab           int64  `xorm:""`                                                   // 固定标签页（1-是 2-否）
	HideChildrenInMenu int64  `xorm:""`                                                   // 子级不展现（1-是 2-否）
	HideInBreadcrumb   int64  `xorm:""`                                                   // 面包屑中不展现（1-是 2-否）
	HideInMenu         int64  `xorm:""`                                                   // 菜单中不展现（1-是 2-否）
	HideInTab          int64  `xorm:""`                                                   // 标签页中不展现（1-是 2-否）
	KeepAlive          int64  `xorm:""`                                                   // 是否缓存（1-是 2-否）
	Sort               int64  `xorm:""`                                                   // 排序
	Icon               string `xorm:""`                                                   // 菜单图标
	Redirect           string `xorm:""`                                                   // 跳转路径
}

func (src *MenusGlobal) CopyTo(dst *MenusGlobal) {
	*dst = *src
}
