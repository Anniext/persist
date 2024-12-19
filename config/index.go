package config

type Index struct {
	Cols            []string
	Keys            string
	ClsPointKeys    string
	CommaKeys       string
	CommaKeyKeys    string
	KeyTypes        string
	Types           []string
	Unique          bool
	Pk              bool
	RBTreeValue     bool
	KeyTypesStripPk string
	KeysStripPk     []string
	EffectIndex     []*Index
}

type TypeInfo struct {
	DSPkg     string // 依赖的ds 包名
	Name      string // 类型名
	IsBool    bool
	IsInt8    bool
	IsInt     bool
	IsFloat   bool
	IsString  bool
	IsPoint   bool
	IsStruct  bool
	Bit       int
	IsPk      bool
	IsNotNull bool
	NeedInit  bool
	//IsMap       bool
}
