// Package data MenusGlobal 使用泛型实现的持久化管理器示例
// 这是一个独立的实现，展示如何使用泛型重构
package data

import (
	"encoding/binary"
	"log"
	"runtime/debug"

	"github.com/spelens-gud/persist/core"
	"github.com/spelens-gud/persist/model"
	"xorm.io/xorm"
)

// MenusGlobalKeyRefactored 主键结构（重构版）
type MenusGlobalKeyRefactored struct {
	AuthId int64
}

// MenusGlobalCompositeKeyRefactored 复合键结构（重构版）
type MenusGlobalCompositeKeyRefactored struct {
	AuthId int64
	Type   string
}

// MenusGlobalBitSetRefactored BitSet结构（重构版）
type MenusGlobalBitSetRefactored struct {
	set [1]uint64 // 根据字段数量计算：19个字段需要1个uint64
}

// MenusGlobal字段索引（重构版）
const (
	EMenusGlobalFieldIndexAuthIdRefactored MenusGlobalFieldIndexRefactored = iota
	EMenusGlobalFieldIndexParentIdRefactored
	EMenusGlobalFieldIndexTreePathRefactored
	EMenusGlobalFieldIndexNameRefactored
	EMenusGlobalFieldIndexTypeRefactored
	EMenusGlobalFieldIndexRouteNameRefactored
	EMenusGlobalFieldIndexPathRefactored
	EMenusGlobalFieldIndexComponentRefactored
	EMenusGlobalFieldIndexPermRefactored
	EMenusGlobalFieldIndexStatusRefactored
	EMenusGlobalFieldIndexAffixTabRefactored
	EMenusGlobalFieldIndexHideChildrenInMenuRefactored
	EMenusGlobalFieldIndexHideInBreadcrumbRefactored
	EMenusGlobalFieldIndexHideInMenuRefactored
	EMenusGlobalFieldIndexHideInTabRefactored
	EMenusGlobalFieldIndexKeepAliveRefactored
	EMenusGlobalFieldIndexSortRefactored
	EMenusGlobalFieldIndexIconRefactored
	EMenusGlobalFieldIndexRedirectRefactored
	EMenusGlobalFiledIndexLengthRefactored
)

type MenusGlobalFieldIndexRefactored = uint

const (
	EMenusGlobalWordSizeRefactored     = MenusGlobalFieldIndexRefactored(64)
	EMenusGlobalLog2WordSizeRefactored = MenusGlobalFieldIndexRefactored(6)
	EMenusGlobalAllBitsRefactored      = uint64(0xffffffffffffffff)
)

// Get 获取位
func (b *MenusGlobalBitSetRefactored) Get(i MenusGlobalFieldIndexRefactored) bool {
	if i >= EMenusGlobalFiledIndexLengthRefactored {
		return false
	}
	return b.set[i>>EMenusGlobalLog2WordSizeRefactored]&(1<<(i&(EMenusGlobalWordSizeRefactored-1))) != 0
}

// Set 设置位
func (b *MenusGlobalBitSetRefactored) Set(i MenusGlobalFieldIndexRefactored) *MenusGlobalBitSetRefactored {
	if i >= EMenusGlobalFiledIndexLengthRefactored {
		return nil
	}
	b.set[i>>EMenusGlobalLog2WordSizeRefactored] |= 1 << (i & (EMenusGlobalWordSizeRefactored - 1))
	return b
}

// Clear 清除位
func (b *MenusGlobalBitSetRefactored) Clear(i MenusGlobalFieldIndexRefactored) *MenusGlobalBitSetRefactored {
	if i >= EMenusGlobalFiledIndexLengthRefactored {
		return b
	}
	b.set[i>>EMenusGlobalLog2WordSizeRefactored] &^= 1 << (i & (EMenusGlobalWordSizeRefactored - 1))
	return b
}

// Merge 合并BitSet
func (b *MenusGlobalBitSetRefactored) Merge(compare MenusGlobalBitSetRefactored) *MenusGlobalBitSetRefactored {
	for i, word := range b.set {
		b.set[i] = word | compare.set[i]
	}
	return b
}

// ClearAll 清除所有位
func (b *MenusGlobalBitSetRefactored) ClearAll() *MenusGlobalBitSetRefactored {
	if b != nil {
		for i := range b.set {
			b.set[i] = 0
		}
	}
	return b
}

// SetAll 设置所有位
func (b *MenusGlobalBitSetRefactored) SetAll() *MenusGlobalBitSetRefactored {
	if b != nil {
		for i := range b.set {
			b.set[i] = EMenusGlobalAllBitsRefactored
		}
	}
	return b
}

// IsSetAll 检查是否所有位都已设置
func (b *MenusGlobalBitSetRefactored) IsSetAll() bool {
	if b != nil {
		for i := range b.set {
			if b.set[i] != EMenusGlobalAllBitsRefactored {
				return false
			}
		}
	}
	return true
}

// MenusGlobalSerializerRefactored 序列化器（重构版）
type MenusGlobalSerializerRefactored struct{}

// ToBytes 序列化MenusGlobal
func (s *MenusGlobalSerializerRefactored) ToBytes(obj *model.MenusGlobal) []byte {
	return s.toBytesWithBitSet(obj, func() MenusGlobalBitSetRefactored {
		var bs MenusGlobalBitSetRefactored
		bs.SetAll()
		return bs
	}())
}

// toBytesWithBitSet 使用BitSet序列化（简化版，仅展示核心逻辑）
func (s *MenusGlobalSerializerRefactored) toBytesWithBitSet(cls *model.MenusGlobal, bitSet MenusGlobalBitSetRefactored) []byte {
	var err error
	if cls == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			log.Println("recovered in ", r)
			log.Println("stack: ", string(debug.Stack()))
		}
		if err != nil {
			log.Println("PersistToBytes Error", err.Error())
		}
	}()

	size := 0

	// 计算大小（简化版，仅展示部分字段）
	if bitSet.Get(EMenusGlobalFieldIndexAuthIdRefactored) {
		size += 1 + 8
	} else {
		size += 1
	}

	if bitSet.Get(EMenusGlobalFieldIndexNameRefactored) {
		size += 1 + 4 + len(cls.Name)
	} else {
		size += 1
	}

	// 序列化
	data := make([]byte, size)
	i := 0

	// AuthId
	if bitSet.Get(EMenusGlobalFieldIndexAuthIdRefactored) {
		data[i] |= core.EMarshalFlagBitSet
		i += 1
		binary.LittleEndian.PutUint64(data[i:], uint64(cls.AuthId))
		i += 8
	} else {
		i += 1
	}

	// Name
	if bitSet.Get(EMenusGlobalFieldIndexNameRefactored) {
		data[i] |= core.EMarshalFlagBitSet
		i += 1
		binary.LittleEndian.PutUint32(data[i:], uint32(len(cls.Name)))
		i += 4
		copy(data[i:], cls.Name)
		i += len(cls.Name)
	} else {
		i += 1
	}

	// 其他字段类似处理...

	return data
}

// FromBytes 反序列化MenusGlobal（简化版）
func (s *MenusGlobalSerializerRefactored) FromBytes(data []byte) *model.MenusGlobal {
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
	cls := &model.MenusGlobal{}

	// AuthId
	if data[i]&core.EMarshalFlagBitSet >= 1 {
		i += 1
		cls.AuthId = int64(binary.LittleEndian.Uint64(data[i:]))
		i += 8
	} else {
		i += 1
	}

	// Name
	if data[i]&core.EMarshalFlagBitSet >= 1 {
		i += 1
		lenField := int(binary.LittleEndian.Uint32(data[i:]))
		i += 4
		cls.Name = string(data[i : i+lenField])
		i += lenField
	} else {
		i += 1
	}

	// 其他字段类似处理...

	return cls
}

// MenusGlobalManagerRefactored 泛型管理器（重构版）
type MenusGlobalManagerRefactored struct {
	*Manager[model.MenusGlobal, MenusGlobalKeyRefactored, MenusGlobalBitSetRefactored]

	// 哈希索引
	hashAuthId     *HashIndex[MenusGlobalKeyRefactored, *model.MenusGlobal]
	hashAuthIdType *MultiHashIndex[MenusGlobalCompositeKeyRefactored, *model.MenusGlobal]
}

// NewMenusGlobalManagerRefactored 创建MenusGlobal泛型管理器
func NewMenusGlobalManagerRefactored(engine *xorm.Engine) *MenusGlobalManagerRefactored {
	var bitSetAll MenusGlobalBitSetRefactored
	bitSetAll.SetAll()

	m := &MenusGlobalManagerRefactored{
		Manager:        NewManager[model.MenusGlobal, MenusGlobalKeyRefactored, MenusGlobalBitSetRefactored](engine, &MenusGlobalSerializerRefactored{}, bitSetAll),
		hashAuthId:     NewHashIndex[MenusGlobalKeyRefactored, *model.MenusGlobal](),
		hashAuthIdType: NewMultiHashIndex[MenusGlobalCompositeKeyRefactored, *model.MenusGlobal](),
	}

	return m
}

// GetByAuthId 通过AuthId获取MenusGlobal
func (m *MenusGlobalManagerRefactored) GetByAuthId(authId int64) (*model.MenusGlobal, bool) {
	key := MenusGlobalKeyRefactored{AuthId: authId}
	return m.hashAuthId.Get(key)
}

// GetByAuthIdType 通过AuthId和Type获取MenusGlobal列表
func (m *MenusGlobalManagerRefactored) GetByAuthIdType(authId int64, typ string) []*model.MenusGlobal {
	key := MenusGlobalCompositeKeyRefactored{AuthId: authId, Type: typ}
	return m.hashAuthIdType.GetAll(key)
}

// Insert 插入MenusGlobal
func (m *MenusGlobalManagerRefactored) Insert(obj *model.MenusGlobal) error {
	// 添加到主索引
	key := MenusGlobalKeyRefactored{AuthId: obj.AuthId}
	m.hashAuthId.Set(key, obj)

	// 添加到复合索引
	compositeKey := MenusGlobalCompositeKeyRefactored{AuthId: obj.AuthId, Type: obj.Type}
	m.hashAuthIdType.Add(compositeKey, obj)

	return nil
}

// Delete 删除MenusGlobal
func (m *MenusGlobalManagerRefactored) Delete(obj *model.MenusGlobal) error {
	// 从主索引删除
	key := MenusGlobalKeyRefactored{AuthId: obj.AuthId}
	m.hashAuthId.Remove(key)

	// 从复合索引删除
	compositeKey := MenusGlobalCompositeKeyRefactored{AuthId: obj.AuthId, Type: obj.Type}
	m.hashAuthIdType.Remove(compositeKey, obj)

	return nil
}

// RangeByAuthIdType 遍历指定AuthId和Type的所有MenusGlobal
func (m *MenusGlobalManagerRefactored) RangeByAuthIdType(authId int64, typ string, f func(*model.MenusGlobal) bool) {
	key := MenusGlobalCompositeKeyRefactored{AuthId: authId, Type: typ}
	m.hashAuthIdType.RangeValues(key, f)
}

// GMenusGlobalManagerRefactored 全局泛型管理器实例（重构版）
var GMenusGlobalManagerRefactored *MenusGlobalManagerRefactored
