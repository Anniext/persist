package config

type ArgsInfo struct {
	Save                bool
	PersistPkgPath      string
	PersistPkgName      string
	UnloadKey           string
	Unload              bool
	WarningFlag         int64
	BombDir             string
	OptimizeFlag        int64
	MaxInsertRows       int64
	QueueThreshold      int64
	QueueEmptySleepTime int64
	FileName            string
	NeedSeparate        bool
	SwitchTable         string
	RBTreeCap           int32
}
