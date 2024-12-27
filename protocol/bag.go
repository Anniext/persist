//go:generate ./persist.exe -src=protocol  -dst=data -pkgName=data -unload=true -fileName=bag.go  ItemLocal
package protocol

type ItemLocal struct {
	Uid      int32 `xorm:"pk" hash:"group=1;unique=1" hash:"group=2;unique=0"`
	ItemId   int32 `xorm:"pk" hash:"group=1;unique=1"`
	ItemNum  int64 `xorm:""`
	ItemTime int64 `xorm:""`
}

func (src *ItemLocal) CopyTo(dst *ItemLocal) {
	*dst = *src
}

type ItemTemp struct {
	ItemId  int32
	ItemNum int64
}
