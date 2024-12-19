package core

import (
	"html/template"
	"path"
	"persist/config"
	tpl "persist/template"
	"strings"
	"sync"

	"github.com/Masterminds/sprig"
)

func GenPersistMethod(wg *sync.WaitGroup, srcDir, goFile, dstDir, goPackage, tableName string, argsInfo *config.ArgsInfo) {
	defer wg.Done()

	tt := template.Must(template.New("Main").Funcs(sprig.TxtFuncMap()).Parse(tpl.Main))
	genFileName := "001_" + strings.ToLower(tableName) + "_persist.go"
	goFilePathAbs := path.Join(srcDir, goFile)
	visitor := util.ParseTable(goFilePathAbs, tableName, argsInfo)
	visitor.PackageName = goPackage
}
