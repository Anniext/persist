package test

import (
	"anniext.asia/xt/utils/log"
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

	log.Error("test")
}
