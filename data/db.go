package data

import (
	"anniext.asia/xt/persist/core"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"sync"
	"time"
	"xorm.io/xorm"
)

/*
 * 数据库
 * @Description:
 */

var gEngine *xorm.Engine
var engineOnce sync.Once

func GetDB() *xorm.Engine {
	engineOnce.Do(func() {
		var err error
		gEngine, err = xorm.NewEngine("mysql", "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&interpolateParams=true")
		if err != nil {
			gEngine = nil
			log.Println("GetDB error", err)
		} else {
			//gEngine.ShowSQL(true)
			gEngine.SetMaxIdleConns(2)            //设置连接池中的保持连接的最大连接数
			gEngine.SetMaxOpenConns(4)            //设置连接池的打开的最大连接数
			gEngine.SetConnMaxLifetime(time.Hour) //设置连接超时时间
			//gEngine.AddHook()
		}
	})
	return gEngine
}

func Register(name string, persist core.IPersist) {
	core.RegisterPersist(name, persist)
}

func Exit() (err error) {
	log.Println("Save Begin")
	core.ExitPersist()
	log.Println("Save End")
	log.Println("SyncData Begin")
	err = core.SyncDataPersist(true)
	if err != nil {
		return
	}
	log.Println("SyncData End")
	return
}

func Run() (err error) {
	log.Println("Run Begin")
	err = core.RunPersist()
	if err != nil {
		return
	}
	log.Println("Run End")
	return nil
}

func Dead() bool {
	return core.DeadPersist()
}

func Init() (err error) {
	engine := GetDB()
	if engine == nil {
		panic("GetDB Error")
	}
	err = core.SyncPersist()
	if err != nil {
		return
	}
	return
}
