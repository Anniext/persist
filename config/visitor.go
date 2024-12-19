package config

import "go/ast"

// MyVisitor indexList [{"cols":[key1, key2], "types":[type1, type2], "unique"=true "pk"=true},... ]
type MyVisitor struct {
	Depth               int
	DataName            string
	PackageName         string
	DataTypeSpec        *ast.TypeSpec
	DataStructType      *ast.StructType
	HashIndexList       []*Index
	RBTreeIndexList     []*Index
	ModifyHashIndexList []*Index
	ModifyColList       []string
	ColEffectHashMap    map[string][]*Index
	KeyTypeMap          map[string]string
	HashIndexUnload     *Index
	HashIndexPk         *Index
	IndexRBTreeValue    *Index
	ArgsInfo            *ArgsInfo
	FieldNameList       []string
	FieldTypeList       []TypeInfo
	DSImportsPackage    map[string]string // ds结构依赖的其他包 map[pkgName]pkgPath
	DSImport            string            // ds结构依赖的包名
}
