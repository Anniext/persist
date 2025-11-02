package persist

const (
	EMenusGlobalManagerStateIdle   = 0 // 初始化
	EMenusGlobalManagerStateNormal = 1 // 正常运行
	EMenusGlobalManagerStatePanic  = 2 // 非法停止

	EMenusGlobalTableStateDisk      = 0 // 导出
	EMenusGlobalTableStateLoading   = 1 // 全导入开始
	EMenusGlobalTableStateMemory    = 2 // 全导入完成
	EMenusGlobalTableStateUnloading = 3 // 正在全导出

	EMenusGlobalLoadStateDisk             = 0 // 不存在 or 导出
	EMenusGlobalLoadStateLoading          = 1 // 导入开始
	EMenusGlobalLoadStateMemory           = 2 // 导入完成
	EMenusGlobalLoadStatePrepareUnloading = 3 // 准备导出
	EMenusGlobalLoadStateUnloading        = 4 // 正在导出

	EMenusGlobalOpInsert = 1 // 新建
	EMenusGlobalOpUpdate = 2 // 修改
	EMenusGlobalOpDelete = 3 // 删除
	EMenusGlobalOpUnload = 4 // 导出

	EMenusGlobalCollectStateNormal    = 0 // 正常
	EMenusGlobalCollectStateSaveSync  = 1 // 开始退出, 清理同步队列
	EMenusGlobalCollectStateSaveCache = 2 // 开始退出,清理缓存队列
	EMenusGlobalCollectStateSaveDone  = 3 // 写回完成

)
