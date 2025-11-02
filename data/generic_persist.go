// Package data 提供泛型持久化框架
package data

import (
	"encoding/base64"
	"encoding/binary"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"xorm.io/xorm"
)

// 持久化管理器状态常量
const (
	EManagerStateIdle   = 0 // 初始化
	EManagerStateNormal = 1 // 正常运行
	EManagerStatePanic  = 2 // 非法停止

	ETableStateDisk      = 0 // 导出
	ETableStateLoading   = 1 // 全导入开始
	ETableStateMemory    = 2 // 全导入完成
	ETableStateUnloading = 3 // 正在全导出

	ELoadStateDisk             = 0 // 不存在 or 导出
	ELoadStateLoading          = 1 // 导入开始
	ELoadStateMemory           = 2 // 导入完成
	ELoadStatePrepareUnloading = 3 // 准备导出
	ELoadStateUnloading        = 4 // 正在导出

	EOpInsert = 1 // 新建
	EOpUpdate = 2 // 修改
	EOpDelete = 3 // 删除
	EOpUnload = 4 // 导出

	ECollectStateNormal    = 0 // 正常
	ECollectStateSaveSync  = 1 // 开始退出, 清理同步队列
	ECollectStateSaveCache = 2 // 开始退出,清理缓存队列
	ECollectStateSaveDone  = 3 // 写回完成
)

// DeepCopy 深拷贝接口，persist对象必须支持并发访问
type DeepCopy[T any] interface {
	CopyTo(t *T)
}

// Overload 未落地数据超过阈值时调用
type Overload interface {
	Overload(queueSize int, lastWriteBackTime time.Duration)
}

// Serializer 序列化接口
type Serializer[T any] interface {
	// ToBytes 将对象序列化为字节数组
	ToBytes(obj *T) []byte
	// FromBytes 从字节数组反序列化对象
	FromBytes(data []byte) *T
}

// PersistSync 同步数据结构
type PersistSync[T any, B any] struct {
	Data   *T
	Op     int8
	BitSet B
}

// Manager 泛型持久化管理器
// T: 数据类型
// K: 主键类型
// B: BitSet类型
type Manager[T any, K comparable, B any] struct {
	// 管理器状态
	managerState int32
	loadAll      int32

	// 对象池和队列
	pool              *sync.Pool
	syncChan          chan *PersistSync[T, B]
	syncQueue         *[]*PersistSync[T, B]
	cacheQueue        *[]*PersistSync[T, B]
	FailQueue         []*PersistSync[T, B]
	InsertQueue       []*PersistSync[T, B]
	lastWriteBackTime time.Duration

	// 同步控制
	syncBegin chan bool
	syncEnd   chan bool
	exitBegin chan bool
	exitEnd   chan bool

	// 数据库引擎
	engine *xorm.Engine

	// 序列化器
	serializer Serializer[T]

	// BitSet 全标记
	bitSetAll B
}

// NewManager 创建泛型管理器
func NewManager[T any, K comparable, B any](
	engine *xorm.Engine,
	serializer Serializer[T],
	bitSetAll B,
) *Manager[T, K, B] {
	m := &Manager[T, K, B]{
		engine:     engine,
		serializer: serializer,
		bitSetAll:  bitSetAll,
	}

	m.syncChan = make(chan *PersistSync[T, B], 16)
	tmpSyncQueue := make([]*PersistSync[T, B], 0)
	m.syncQueue = &tmpSyncQueue
	m.syncEnd = make(chan bool)
	m.syncBegin = make(chan bool)
	m.exitBegin = make(chan bool)
	m.exitEnd = make(chan bool)
	tmpCacheQueue := make([]*PersistSync[T, B], 0)
	m.cacheQueue = &tmpCacheQueue
	m.lastWriteBackTime = 1 * time.Millisecond
	m.pool = &sync.Pool{New: func() interface{} { var t T; return &t }}

	return m
}

// Run 运行管理器
func (m *Manager[T, K, B]) Run() error {
	if atomic.CompareAndSwapInt32(&m.managerState, EManagerStateIdle, EManagerStateNormal) {
		go m.Collect()
	} else if atomic.CompareAndSwapInt32(&m.managerState, EManagerStatePanic, EManagerStateNormal) {
		go m.Collect()
	}
	return nil
}

// Dead 管理器是否出错
func (m *Manager[T, K, B]) Dead() bool {
	return atomic.LoadInt32(&m.managerState) != EManagerStateNormal
}

// Collect 收集并处理同步数据
func (m *Manager[T, K, B]) Collect() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Collect panic:", r)
			log.Println("stack:", string(debug.Stack()))
			atomic.StoreInt32(&m.managerState, EManagerStatePanic)
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case sync := <-m.syncChan:
			*m.syncQueue = append(*m.syncQueue, sync)
		case <-ticker.C:
			if len(*m.syncQueue) > 0 {
				m.processSyncQueue()
			}
		case <-m.exitBegin:
			m.shutdown()
			return
		}
	}
}

// processSyncQueue 处理同步队列
func (m *Manager[T, K, B]) processSyncQueue() {
	// 交换队列
	oldQueue := m.syncQueue
	newQueue := make([]*PersistSync[T, B], 0)
	m.syncQueue = &newQueue

	// 处理旧队列
	for _, sync := range *oldQueue {
		// 这里应该调用实际的数据库操作
		_ = sync
	}
}

// shutdown 关闭管理器
func (m *Manager[T, K, B]) shutdown() {
	// 处理剩余数据
	m.processSyncQueue()
	m.exitEnd <- true
}

// BytesToPersist 反序列化
func (m *Manager[T, K, B]) BytesToPersist(data []byte) *T {
	if data == nil {
		return nil
	}
	return m.serializer.FromBytes(data)
}

// PersistToBytes 序列化
func (m *Manager[T, K, B]) PersistToBytes(obj *T) []byte {
	if obj == nil {
		return nil
	}
	return m.serializer.ToBytes(obj)
}

// BytesToPersistSync 反序列化sync
func (m *Manager[T, K, B]) BytesToPersistSync(data []byte) *PersistSync[T, B] {
	if data == nil {
		return nil
	}
	// 实现反序列化逻辑
	return nil
}

// PersistSyncToBytes 序列化sync
func (m *Manager[T, K, B]) PersistSyncToBytes(sync *PersistSync[T, B]) []byte {
	if sync == nil {
		return nil
	}
	// 实现序列化逻辑
	return nil
}

// StringToPersistSync 从字符串反序列化sync
func (m *Manager[T, K, B]) StringToPersistSync(data string) *PersistSync[T, B] {
	if data == "" {
		return nil
	}
	buf, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil
	}
	return m.BytesToPersistSync(buf)
}

// PersistSyncToString 序列化sync到字符串
func (m *Manager[T, K, B]) PersistSyncToString(sync *PersistSync[T, B]) string {
	buf := m.PersistSyncToBytes(sync)
	if buf == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// UnmarshalFailQueue 失败队列反序列化
func (m *Manager[T, K, B]) UnmarshalFailQueue(data []byte, failQueue *[]*PersistSync[T, B]) error {
	if data == nil || failQueue == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			log.Println("UnmarshalFailQueue panic:", r)
			log.Println("stack:", string(debug.Stack()))
		}
	}()

	i := 0
	lenFailQueue := binary.LittleEndian.Uint32(data[i:])
	i += 4
	*failQueue = make([]*PersistSync[T, B], lenFailQueue)
	for idx := 0; idx < int(lenFailQueue); idx++ {
		lenSyncData := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		sync := m.BytesToPersistSync(data[i : i+lenSyncData])
		i += lenSyncData
		(*failQueue)[idx] = sync
	}
	return nil
}

// MarshalFailQueue 失败队列序列化
func (m *Manager[T, K, B]) MarshalFailQueue(failQueue []*PersistSync[T, B]) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("MarshalFailQueue panic:", r)
			log.Println("stack:", string(debug.Stack()))
		}
	}()

	syncDataList := make([][]byte, len(failQueue))
	size := 4
	for idx := range failQueue {
		pData := m.PersistSyncToBytes(failQueue[idx])
		syncDataList[idx] = pData
		size += 4 + len(pData)
	}

	data := make([]byte, size)
	i := 0
	binary.LittleEndian.PutUint32(data[i:], uint32(len(failQueue)))
	i += 4
	for idx := range failQueue {
		binary.LittleEndian.PutUint32(data[i:], uint32(len(syncDataList[idx])))
		i += 4
		copy(data[i:], syncDataList[idx])
		i += len(syncDataList[idx])
	}
	return data, nil
}

// acquireDeepCopyObject 获取深拷贝对象
func (m *Manager[T, K, B]) acquireDeepCopyObject(obj *T) *T {
	if v, ok := any(obj).(DeepCopy[T]); ok {
		ret := new(T)
		v.CopyTo(ret)
		return ret
	}
	// 使用序列化方式深拷贝
	return m.BytesToPersist(m.PersistToBytes(obj))
}

// Exit 退出管理器
func (m *Manager[T, K, B]) Exit() error {
	m.exitBegin <- true
	<-m.exitEnd
	return nil
}
