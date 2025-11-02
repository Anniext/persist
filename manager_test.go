package persist

type GUserGlobal struct {
	Uid      int64
	Name     string
	Password string
}

func (g *GUserGlobal) isModel() {
}

type GUserPrimaryId struct {
	Uid int64
}

func (g *GUserPrimaryId) isPrimaryId() {}

type GUserCompoundPrimarystruct struct {
	Uid  int64
	Name string
}

func (g *GUserCompoundPrimarystruct) isCompoundPrimaryId() {}
