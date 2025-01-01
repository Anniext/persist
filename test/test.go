package main

import (
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
	if err != nil {
		log.Panic(err.Error())

	}
	data.GGoodsLocalManager.LoadAll()

	data.GGoodsLocalManager.NewGoodsLocal(&protocol.GoodsLocal{
		Uid:  1,
		Time: 0,
		Name: "sss",
	})
	if err != nil {
		log.Error(err)
	}

	uid := data.GGoodsLocalManager.GetGoodsLocalByUid(1)

	log.Info(uid.Name)
	time.Sleep(time.Second * 10)
}
