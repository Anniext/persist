package persist

import (
	"encoding/binary"
	"log"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	persistCore "github.com/spelens-gud/persist/core"
	"github.com/spelens-gud/persist/model"
	"xorm.io/xorm"
)

var GlobalDBFiledMap sync.Map
var GGlobalManager sync.Map

// GlobalKeyTypeHashPrimaryId 主键接口
type GlobalKeyTypeHashPrimaryId interface {
	isPrimaryId() int64
}

// GlobalKeyTypeHashCompoundPrimaryId 复合主键接口
// 实现此接口的类型必须是 comparable，以便可以作为 map 的 key
type GlobalKeyTypeHashCompoundPrimaryId interface {
	isCompoundPrimaryId()
}

// GlobalCompoundKeyProvider 提供复合主键的接口
// T 是模型类型，K 是复合主键类型
type GlobalCompoundKeyProvider[K GlobalKeyTypeHashCompoundPrimaryId] interface {
	// GetCompoundKey 获取对象的复合主键
	GetCompoundKey() K
}

// GlobalDeepCopy persist对象必须支持并发访问, 不实现该接口默认深拷贝对象 (1 建议实现该接口,反射效率较低  2 map 建议生成syncmap  3 slice 建议深拷贝)
type GlobalDeepCopy[T GlobalModel] interface {
	CopyTo(t T)
}

// GlobalModel 全局模型接口
type GlobalModel interface {
	GlobalKeyTypeHashPrimaryId
	comparable
	isModel()
}

// GlobalManager 全局管理器
// T 是模型类型，K 是复合主键类型
type GlobalManager[T GlobalModel, K GlobalKeyTypeHashCompoundPrimaryId] struct {
	managerState int32 // 0:初始化  1:正常运行  2:非法停止
	loadAll      int32 // 0:导出  1:全导入开始  2:全导入完成  3:正在全导出

	pool *sync.Pool // 对象池

	syncChan    chan *GlobalSync[T] // 同步chan
	syncQueue   *[]*GlobalSync[T]   // 同步队列
	cacheQueue  *[]*GlobalSync[T]   // 缓存队列
	FailQueue   []*GlobalSync[T]    // 失败队列
	InsertQueue []*GlobalSync[T]    // 插入队列

	lastWriteBackTime time.Duration // 最后一次写入时间

	syncBegin chan bool // 同步开始
	syncEnd   chan bool // 同步结束
	exitBegin chan bool // 退出开始
	exitEnd   chan bool // 退出结束

	hasPrimaryId         PrimarySyncMap[T]                       // key: 主键, value: 对象
	hasCompoundPrimaryId CompoundPrimarySyncMap[GlobalBitSet[T]] // key: K (复合主键), value: *SetSyncMap[T]

	engine *xorm.Engine

	bitSetAll GlobalBitSet[T] // 全局bitSet

	// compoundKeyExtractor 复合主键提取函数（可选）
	compoundKeyExtractor func(T) K
}

// NewGlobalManager 创建全局管理器
// keyExtractor: 可选的复合主键提取函数，如果 T 实现了 GlobalCompoundKeyProvider[K] 接口则可以为 nil
func NewGlobalManager[T GlobalModel, K GlobalKeyTypeHashCompoundPrimaryId](
	engine *xorm.Engine,
	keyExtractor func(T) K,
) (m *GlobalManager[T, K]) {
	m = &GlobalManager[T, K]{
		engine:               engine,
		compoundKeyExtractor: keyExtractor,
	}

	m.syncChan = make(chan *GlobalSync[T], runtime.NumCPU()*2)
	tmpSyncQueue := make([]*GlobalSync[T], 0)
	m.syncQueue = &tmpSyncQueue
	m.syncEnd = make(chan bool)
	m.syncBegin = make(chan bool)
	m.exitBegin = make(chan bool)
	m.exitEnd = make(chan bool)
	tmpCacheQueue := make([]*GlobalSync[T], 0)
	m.cacheQueue = &tmpCacheQueue
	m.lastWriteBackTime = 1 * time.Millisecond
	m.pool = &sync.Pool{New: func() interface{} { return &model.MenusGlobal{} }}

	m.bitSetAll.SetAll()

	if engine != nil {
		m.injectDBField()
	}

	return
}

// injectDBField 注入db field 进入缓存
func (m *GlobalManager[T, K]) injectDBField() {
	var v T
	typOf := reflect.TypeOf(v)
	if metas, ok := cache.Load(typOf); !ok {
		metasNames := metas.(*meta)
		dbFieldList := make([]string, len(metasNames.names))
		for idx, name := range metasNames.names {
			dbFieldList[idx] = m.engine.GetColumnMapper().Obj2Table(name)
		}
		GlobalDBFiledMap.LoadOrStore(typOf, dbFieldList)
	}
}

// NewGlobal 通过主键创建或获取对象
// 注意：由于需要指定复合主键类型 K，建议在具体实现中创建专门的函数
// 例如：NewMenusGlobal(primaryId int64) *MenusGlobal
func NewGlobal[T GlobalModel, K GlobalKeyTypeHashCompoundPrimaryId](primaryId int64) (ormCls T) {
	globalModel := getGlobalManager[T, K]()
	if globalModel == nil {
		return
	}
	ormCls = globalModel.GetGlobalByPrimaryId(primaryId)
	var zero T
	if ormCls != zero {
		return
	}
	//ormCls, _ = globalModel.NewGlobal(&T{primaryId})
	return
}

func getGlobalManager[T GlobalModel, K GlobalKeyTypeHashCompoundPrimaryId]() *GlobalManager[T, K] {
	var v T
	typOf := reflect.TypeOf(v)
	globalManager, ok := GGlobalManager.Load(typOf)
	if !ok {
		return nil
	}
	return globalManager.(*GlobalManager[T, K])
}

// NewGlobal 添加对象并异步写回数据库, (1 数据没有导入或已经导出, 2 数据已存在, 3 对象为空) 会返回失败
func (m *GlobalManager[T, K]) NewGlobal(cls T) (data T, err error) {

	if cls == nil {
		return data, persistCore.EPersistErrorNil
	}

	if m.LoadAllState() != EMenusGlobalLoadStateMemory {
		return data, persistCore.EPersistErrorNotInMemory
	}

	actual, success := m.addGlobal(cls)

	if success {
		//m.InitDS(cls)
		bitSet := GlobalBitSet[T]{}
		bitSet.SetAll()
		newCls := m.acquireDeepCopyObject(cls)

		persistSync := &MenusGlobalSync{Data: newCls, Op: EMenusGlobalOpInsert, BitSet: bitSet}

		log.Println("[sql trace MenusGlobal]", m.PersistSyncToString(persistSync))

		m.syncChan <- persistSync

	} else {
		return actual, persistCore.EPersistErrorAlreadyExist
	}

	return actual, nil
}

type PrimaryKeyTypeHashPrimaryId int64

func (PrimaryKeyTypeHashPrimaryId) isPrimaryId() int64 { return 0 }

// GetGlobalByPrimaryId 通过主键查找对象
func (m *GlobalManager[T, K]) GetGlobalByPrimaryId(primaryId int64) (data T) {
	if data, ok := m.hasPrimaryId.Load(PrimaryKeyTypeHashPrimaryId(primaryId)); ok {
		return data
	}
	return data
}

// LoadAllState 所有数据导入状态
func (m *GlobalManager[T, K]) LoadAllState() int32 {
	return atomic.LoadInt32(&m.loadAll)
}

// getCompoundKey 获取对象的复合主键
// 优先使用接口方法，其次使用提取函数
func (m *GlobalManager[T, K]) getCompoundKey(cls T) (key K, ok bool) {
	// 方式1: 检查是否实现了 GlobalCompoundKeyProvider 接口
	if provider, ok := any(cls).(GlobalCompoundKeyProvider[K]); ok {
		return provider.GetCompoundKey(), true
	}

	// 方式2: 使用提取函数
	if m.compoundKeyExtractor != nil {
		return m.compoundKeyExtractor(cls), true
	}

	// 没有提供复合主键的方式
	var zero K
	return zero, false
}

// addGlobal 添加一个对象
func (m *GlobalManager[T, K]) addGlobal(cls T) (data T, ok bool) {
	actual, loaded := m.hasPrimaryId.LoadOrStore(PrimaryKeyTypeHashPrimaryId(cls.isPrimaryId()), cls)
	if !loaded {
		actual = cls

		// 处理复合主键索引
		if compoundKey, hasKey := m.getCompoundKey(cls); hasKey {
			// 使用 sync.Map 存储复合主键索引
			setInterface, _ := m.hasCompoundPrimaryId.LoadOrStore(compoundKey, &SetSyncMap[T]{})
			if set, ok := setInterface.(*SetSyncMap[T]); ok {
				set.Store(cls, true)
			}
		}
	}
	return actual, !loaded
}

// acquireDeepCopyObject 拷贝一个新对象用于写回
func (m *GlobalManager[T, K]) acquireDeepCopyObject(cls T) (ret T) {
	if v, ok := ((interface{})(cls)).(GlobalDeepCopy[T]); ok {
		//ret = m.pool.Get().(*model.MenusGlobal)
		ret = *new(T)
		v.CopyTo(ret)
	} else {
		ret = m.BytesToPersist(m.PersistToBytes(cls, m.bitSetAll))
	}
	return
}

// BytesToPersist反序列化
func (m *GlobalManager[T, K]) BytesToPersist(data []byte) (cls *model.MenusGlobal) {
	var err error
	if data == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			log.Println("recovered in ", r)
			log.Println("stack: ", string(debug.Stack()))
		}
		if err != nil {
			log.Println("BytesToPersist Error", err.Error())
		}
	}()
	i := 0
	cls = &model.MenusGlobal{}

	//AuthId	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.AuthId = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//ParentId	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.ParentId = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//TreePath	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataTreePath := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.TreePath = string(data[i : i+lenFieldDataTreePath])
		i += lenFieldDataTreePath
	} else {
		i += 1
	}

	//Name	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataName := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Name = string(data[i : i+lenFieldDataName])
		i += lenFieldDataName
	} else {
		i += 1
	}

	//Type	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataType := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Type = string(data[i : i+lenFieldDataType])
		i += lenFieldDataType
	} else {
		i += 1
	}

	//RouteName	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataRouteName := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.RouteName = string(data[i : i+lenFieldDataRouteName])
		i += lenFieldDataRouteName
	} else {
		i += 1
	}

	//Path	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataPath := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Path = string(data[i : i+lenFieldDataPath])
		i += lenFieldDataPath
	} else {
		i += 1
	}

	//Component	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataComponent := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Component = string(data[i : i+lenFieldDataComponent])
		i += lenFieldDataComponent
	} else {
		i += 1
	}

	//Perm	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataPerm := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Perm = string(data[i : i+lenFieldDataPerm])
		i += lenFieldDataPerm
	} else {
		i += 1
	}

	//Status	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.Status = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//AffixTab	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.AffixTab = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//HideChildrenInMenu	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.HideChildrenInMenu = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//HideInBreadcrumb	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.HideInBreadcrumb = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//HideInMenu	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.HideInMenu = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//HideInTab	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.HideInTab = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//KeepAlive	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.KeepAlive = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//Sort	int64

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		cls.Sort = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 64 / 8
	} else {
		i += 1
	}

	//Icon	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataIcon := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Icon = string(data[i : i+lenFieldDataIcon])
		i += lenFieldDataIcon
	} else {
		i += 1
	}

	//Redirect	string

	if data[i]&persistCore.EMarshalFlagBitSet >= 1 {
		i += 1

		lenFieldDataRedirect := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Redirect = string(data[i : i+lenFieldDataRedirect])
		i += lenFieldDataRedirect
	} else {
		i += 1
	}

	return
}
