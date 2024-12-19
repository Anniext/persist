package global

const (
	// EOptimizeFlagIndexMutex 保证所有的索引值完全一致, 不开启则保证最终一致性(不建议开启!!! 串行化所有的索引操作, 影响并发量)
	EOptimizeFlagIndexMutex = 1 << iota
	// EOptimizeFlagSQLMerge 保证数据库索引关系完全一致, 不开启则保证最终一致性(建议开启!!! 内存中模拟合并, 性能影响不大, 减轻数据库开销)
	EOptimizeFlagSQLMerge
	// EOptimizeFlagGenerateCode 使用生成代码取代反射, 固定的模型写法, 不保证所有情况正确(强制开启!!! 解决20%的反射开销, 生成代码不一定考虑到了所有情况, 特殊写法需要先测试)
	EOptimizeFlagGenerateCode
	// EOptimizeFlagSQLInsertMerge 合并插入语句, 减少数据库IO, 减少反射次数(建议开启!!! 插入语句越多, 性能提升越明显, 插入非常少,可以不开启,减少cpu消耗)
	EOptimizeFlagSQLInsertMerge
	// EOptimizeFlagTraceSwitch 数据变化追踪开关(酌情开启!!! 大量日志影响性能，开启时请使用MarkUpdateByBitSet不要用MarkUpdate， 减少序列化开销)
	EOptimizeFlagTraceSwitch
	// EOptimizeFlagUsePoolAndDisableDeleteUnload 使用对象池并禁用删除接口(特殊情况开启!!! 行数大于5M且没有删除行为开启, 承降低载, 减少GC耗时, 减少内存)
	EOptimizeFlagUsePoolAndDisableDeleteUnload
	EOptimizeFlagMax
)
