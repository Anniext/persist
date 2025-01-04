package test

import (
	"anniext.asia/xt/persist/core"
	"anniext.asia/xt/persist/data"
	"anniext.asia/xt/persist/protocol"
	"anniext.asia/xt/utils/log"
	"time"
)

func main() {
	slogConfig := log.NewSlogOption(
		log.WithFilenameOption("./tmp/test.log"),
		log.WithMaxSizeOption(10),
		log.WithMaxBackupsOption(10),
		log.WithMaxAgeOption(10),
		log.WithCompressOption(true),
		log.WithProdLevelOption("info"),
		log.WithModeOption("prod"),
	)

	log.NewSlogCore(slogConfig)

	err := data.Init()
	if err != nil {
		log.Panic(err.Error())
	}
	err = data.Run()

	core.SetLoadState2Memory(1)

	data.GItemLocalManager.NewItemLocal(&protocol.ItemLocal{
		Uid:      1,
		ItemId:   2,
		ItemNum:  3,
		ItemTime: 4,
	})

	a := data.GItemLocalManager.GetAll()
	for _, i := range a {
		log.Info(i.ItemId)
	}
	time.Sleep(3 * time.Second)
}
