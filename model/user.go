package model

const (
	RoleLocalAccountIsNotBan = 1 // 用户账号没被ban
	RoleLocalAccountIsBan    = 2 // 用户账号被ban
)

// UserShare 用户共享数据
type UserShare struct {
	Uid           int64  `xorm:"pk" hash:"group=1;unique=1"`
	UserName      string `xorm:"" hash:"group=3;unique=1"` // 用户名
	NickName      string `xorm:""`                         // 昵称
	Password      string `xorm:""`                         // 密码
	Mobile        string `xorm:"" hash:"group=2;unique=1"` // 绑定手机,使用手机登录
	Gender        int32  `xorm:""`                         // 性别 性别(1-男 2-女 0-保密)
	Email         string `xorm:""`                         // 邮箱
	Avatar        string `xorm:""`                         // 头像
	Status        int64  `xorm:"" hash:"group=3;unique=1"` // 状态 1:正常,2:禁用
	DeptId        int64  `xorm:""`                         // 部门id
	RoleId        int64  `xorm:""`                         // 角色id
	Token         string `xorm:""`                         // 密钥
	Remark        string `xorm:""`                         // 备注
	CreateBy      int64  `xorm:""`                         // 创建者ID
	UpdateBy      int64  `xorm:""`                         // 更新者ID
	LastLoginTime int64  `xorm:""`                         // 最后一次登录的时间
	LastLoginIp   string `xorm:""`                         // 最后一次登录的IP
}

func (src *UserShare) CopyTo(dst *UserShare) {
	*dst = *src
}
