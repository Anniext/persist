package core

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"persist/config"
	"runtime/debug"
	"strings"
)

func ParseTable(filePathAbs string, tableName string, argsInfo *config.ArgsInfo) (visitor *config.MyVisitor) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("ParseTable panic", tableName, r)
			fmt.Println("stack: ", string(debug.Stack()))
		}
	}()

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, filePathAbs, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Println(err.Error())
	}
	//ast.Print(fs, f)
	visitor = &config.MyVisitor{}
	visitor.DataName = tableName
	visitor.ArgsInfo = argsInfo
	visitor.DSImportsPackage = make(map[string]string)

	ast.Walk(visitor, f)

	//fmt.Println(visitor)

	indexDict := make(map[string]*Index)
	rbtreeIndexDict := make(map[string]*Index)
	var keyNameList []string
	var rbtreeKeyNameList []string

	visitor.KeyTypeMap = make(map[string]string)
	visitor.ColEffectHashMap = make(map[string][]*Index)

	// 提取所有列名 和 类型
	for _, field := range visitor.DataStructType.Fields.List {
		if IsIgnoreField(field) {
			continue
		}
		visitor.FieldNameList = append(visitor.FieldNameList, GetFieldName(field))
		visitor.FieldTypeList = append(visitor.FieldTypeList, GetFieldTypeInfo(field))
		//fmt.Println(GetFieldTypeInfo(field))
	}
	dsPkgMap := map[string]bool{}
	for _, info := range visitor.FieldTypeList {
		if (info.DSPkg != "") && info.IsNotNull {
			if _, exist := dsPkgMap[info.DSPkg]; exist {
				continue
			}
			dsPkgMap[info.DSPkg] = true
			visitor.DSImport += visitor.DSImportsPackage[info.DSPkg]
			visitor.DSImport += "\n"
		}
	}

	//for i, info := range visitor.FieldTypeList {
	//	if info.IsNotNull {
	//		dsImportMap[]
	//		info.
	//	}
	//
	//}

	// 遍历所有的tag, 提取索引信息
	for _, field := range GetStructBuildInTypeFields(visitor.DataStructType) {
		if _, ok := field.Type.(*ast.Ident); !ok {
			continue
		}
		//fmt.Println(GetFieldName(field))
		//fmt.Println(GetFieldType(field))
		//fmt.Println(GetFieldTags(field))
		var isPk bool
		var isUnique bool
		var index *Index
		var ok bool
		for _, tag := range GetFieldTags(field).Tags() {
			if tag.Key == "xorm" && strings.Contains(tag.Name, "pk") {
				isPk = true
			}
		}

		for _, tag := range GetFieldTags(field).Tags() {
			index = nil
			keyName := tag.Key
			if strings.Contains(tag.Key, "hash") {
				items := strings.Split(tag.Name, ";")
				for _, item := range items {
					kvs := strings.Split(item, "=")
					switch kvs[0] {
					case "unique":
						isUnique = kvs[1] == "1"
					case "pk":
						isPk = kvs[1] == "1"
					case "group":
						keyName += kvs[1]
					}
				}
				if index, ok = indexDict[keyName]; !ok {
					keyNameList = append(keyNameList, keyName)
					index = &Index{}
					indexDict[keyName] = index
				}
				if _, ok = visitor.KeyTypeMap[GetFieldName(field)]; !ok {
					visitor.KeyTypeMap[GetFieldName(field)] = GetFieldType(field)
					visitor.ModifyColList = append(visitor.ModifyColList, GetFieldName(field))
				}

				index.Cols = append(index.Cols, GetFieldName(field))
				index.Keys += GetFieldName(field)
				index.ClsPointKeys += "cls." + GetFieldName(field) + ","
				index.CommaKeys += GetFieldName(field) + ","
				index.CommaKeyKeys += GetFieldName(field) + ": " + GetFieldName(field) + ","
				index.KeyTypes += GetFieldName(field) + " " + GetFieldType(field) + ","
				index.Types = append(index.Types, GetFieldType(field))
				if index != nil {
					index.Pk = isPk
					index.Unique = isUnique
				}
			} else {
			}
			if strings.Contains(tag.Key, "rbtree") {

				var rbtreeIndex *Index
				if rbtreeIndex, ok = rbtreeIndexDict[keyName]; !ok {
					rbtreeKeyNameList = append(rbtreeKeyNameList, keyName)
					rbtreeIndex = &Index{}
					rbtreeIndexDict[keyName] = rbtreeIndex
				}

				items := strings.Split(tag.Name, ";")
				isRBTreeValue := false
				for _, item := range items {
					//kvs := strings.Split(item, "=")
					switch item {
					case "value":
						isRBTreeValue = true
					case "v":
						isRBTreeValue = true
					}
				}
				if _, ok = visitor.KeyTypeMap[GetFieldName(field)]; !ok {
					visitor.KeyTypeMap[GetFieldName(field)] = GetFieldType(field)
					visitor.ModifyColList = append(visitor.ModifyColList, GetFieldName(field))
				}

				rbtreeIndex.Cols = append(rbtreeIndex.Cols, GetFieldName(field))
				rbtreeIndex.Keys += GetFieldName(field)
				rbtreeIndex.ClsPointKeys += "cls." + GetFieldName(field) + ","
				rbtreeIndex.CommaKeys += GetFieldName(field) + ","
				rbtreeIndex.CommaKeyKeys += GetFieldName(field) + ": " + GetFieldName(field) + ","
				rbtreeIndex.KeyTypes += GetFieldName(field) + " " + GetFieldType(field) + ","
				rbtreeIndex.Types = append(rbtreeIndex.Types, GetFieldType(field))
				if rbtreeIndex != nil {
					rbtreeIndex.RBTreeValue = isRBTreeValue
				}
			}
			//fmt.Println(tag.Key, "@", tag.Name)
		}
	}
	//fmt.Println(keyNameList)

	// 主键强制放到第一个
	for _, keyName := range keyNameList {
		index := indexDict[keyName]
		// 记录主键信息
		if index.Pk && index.Unique {
			if visitor.HashIndexPk == nil {
				visitor.HashIndexPk = index
				// 主键移除修改列表
				for _, col := range index.Cols {
					for i := range visitor.ModifyColList {
						if visitor.ModifyColList[i] == col {
							visitor.ModifyColList = append(visitor.ModifyColList[:i], visitor.ModifyColList[i+1:]...)
							break
						}
					}
				}
			} else {
				panic("repeated pk " + tableName + visitor.HashIndexPk.String() + " " + index.String())
			}
		} else {
			// 处理非主键的索引修改
			visitor.HashIndexList = append(visitor.HashIndexList, index)
		}

	}

	for _, rbtreeKeyName := range rbtreeKeyNameList {
		rbtreeIndex := rbtreeIndexDict[rbtreeKeyName]
		if rbtreeIndex.RBTreeValue {
			// 是红黑树的value
			if visitor.IndexRBTreeValue == nil {
				visitor.IndexRBTreeValue = rbtreeIndex
				visitor.RBTreeIndexList = append(visitor.RBTreeIndexList, rbtreeIndex)
			}
		}
	}

	funcNotInIndex := func(col string, index *Index) bool {
		exist := false
		for _, colPk := range index.Cols {
			if col == colPk {
				exist = true
			}
		}
		return !exist
	}

	for _, index := range visitor.HashIndexList {
		notInPk := false
		for idx, col := range index.Cols {
			if funcNotInIndex(col, visitor.HashIndexPk) {
				index.KeyTypesStripPk += col + " " + index.Types[idx] + ","
				index.KeysStripPk = append(index.KeysStripPk, col)
				notInPk = true
			} else {
			}
		}
		if notInPk {
			if len(index.Cols) == 1 {
				for i := range visitor.ModifyColList {
					if visitor.ModifyColList[i] == index.Cols[0] {
						visitor.ModifyColList = append(visitor.ModifyColList[:i], visitor.ModifyColList[i+1:]...)
						break
					}
				}
			}
			visitor.ModifyHashIndexList = append(visitor.ModifyHashIndexList, index)
		}
	}
	if visitor.HashIndexPk != nil {
		visitor.HashIndexList = append([]*Index{visitor.HashIndexPk}, visitor.HashIndexList...)
	}

	for _, index := range visitor.HashIndexList {
		for _, otherIndex := range visitor.HashIndexList {
			for _, col := range index.KeysStripPk {
				if !funcNotInIndex(col, otherIndex) {
					index.EffectIndex = append(index.EffectIndex, otherIndex)
					break
				}
			}
		}
	}
	for _, col := range visitor.ModifyColList {
		for _, otherIndex := range visitor.HashIndexList {
			if !funcNotInIndex(col, otherIndex) {
				visitor.ColEffectHashMap[col] = append(visitor.ColEffectHashMap[col], otherIndex)
			}
		}
	}

	//// 主键强制放到第一个
	//for _, v := range indexDict {
	//	if v.Pk && v.Unique {
	//		visitor.HashIndexList = append(visitor.HashIndexList, v)
	//	}
	//}
	//for _, v := range indexDict {
	//	if !(v.Pk && v.Unique) {
	//		visitor.HashIndexList = append(visitor.HashIndexList, v)
	//	}
	//}

	// 查找是否存在Unload key索引
	if argsInfo.Unload {
		for _, keyName := range keyNameList {
			index := indexDict[keyName]
			// 必须存在数据库索引, 不然导入数据很慢
			if len(index.Cols) == 1 && index.Cols[0] == argsInfo.UnloadKey {
				if visitor.HashIndexUnload != nil {
					panic("repeated unload key. " + tableName + index.String())
				}
				visitor.HashIndexUnload = index
			} else {
				// 暂不支持联合键, 用作导出
			}
		}
	}
	if argsInfo.Unload && ValidWarningFlag(argsInfo.WarningFlag, EWarningFlagUnloadFunction) && visitor.HashIndexUnload == nil {
		fmt.Println("warring: cannot be generated method load. " + tableName + argsInfo.UnloadKey)
		//for _, v := range unloadWarringIndexList {
		//	fmt.Println("warring: cannot be generated method load. " + tableName + v.String())
		//}
	}
	//fmt.Println(visitor.HashIndexUnload)
	//	Uid
	//	int32
	//xorm:"pk" hash:"group=1;unique=0" hash:"group=2;unique=1"
	//	Id
	//	int32
	//xorm:"pk" hash:"group=2;unique=1"

	//for _, field := range GetStructBuildOutTypeFields(visitor.DataStructType) {
	//	block := GetFiledDeepCopyString(field)
	//	fmt.Println("##########",block)
	//}

	return
}
