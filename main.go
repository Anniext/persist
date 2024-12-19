package main

import (
	"fmt"
	"math"
	"os"
	"persist/config"
	"persist/core"
	"persist/global"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	greetingBanner = `
`
)

var (
	NeedChangeDataBase bool
	SwitchTable        string

	save                bool
	srcDir              string
	dstDir              string
	fileName            string
	pkgName             string
	unloadKey           string
	unload              bool
	warningFlag         int64
	maxInsertRows       int64
	queueThreshold      int64
	queueEmptySleepTime int64
	rbtreeCap           int64
	optimizeFlag        int64

	defaultOptimizeFlag int64 = (0 & global.EOptimizeFlagIndexMutex) |
		global.EOptimizeFlagSQLMerge |
		global.EOptimizeFlagGenerateCode |
		global.EOptimizeFlagSQLInsertMerge |
		global.EOptimizeFlagTraceSwitch |
		(0 & global.EOptimizeFlagUsePoolAndDisableDeleteUnload) |
		global.EOptimizeFlagMax

	rootCmd = &cobra.Command{}
)

func init() {
	err := setFlags()
	if err != nil {
		panic(err)
	}
}

func findFirstFromRight(s, src string) int {
	if src == "" {
		return 0 // 如果src是空字符串，则认为它在任何位置都是匹配的
	}

	// 计算src的长度，用于后续比较
	srcLen := len(src)

	// 从s的末尾开始遍历
	for i := len(s) - srcLen; i >= 0; i-- {
		// 检查从i开始的后缀是否与src相等
		if s[i:i+srcLen] == src {
			return i // 返回src开始的位置
		}
	}

	// 如果没有找到，返回-1
	return -1
}

func setFlags() error {
	rootCmd.PersistentFlags().BoolVar(&save, "save", true, "save data to mysql")
	rootCmd.PersistentFlags().StringVar(&srcDir, "src", "", "persist struct path")
	rootCmd.PersistentFlags().StringVar(&dstDir, "dst", "", "generate file path")
	rootCmd.PersistentFlags().StringVar(&fileName, "fileName", "", "persist file name")
	rootCmd.PersistentFlags().StringVar(&pkgName, "pkgName", "", "generate package name")
	rootCmd.PersistentFlags().StringVar(&unloadKey, "unloadKey", "Uid", "unload key name")
	rootCmd.PersistentFlags().BoolVar(&unload, "unload", false, "can be unloaded")
	rootCmd.PersistentFlags().Int64Var(&warningFlag, "warningFlag", math.MaxInt32, "warning flag")
	rootCmd.PersistentFlags().Int64Var(&maxInsertRows, "maxInsertRows", 100, "max rows")
	rootCmd.PersistentFlags().Int64Var(&queueThreshold, "queueThreshold", 10000, "sync queue threshold")
	rootCmd.PersistentFlags().Int64Var(&queueEmptySleepTime, "queueEmptySleepTime", 100, "sleep time millisecond")
	rootCmd.PersistentFlags().BoolVar(&NeedChangeDataBase, "separate", false, "Persist need separate database")
	rootCmd.PersistentFlags().Int64Var(&rbtreeCap, "rbtreeCap", math.MaxInt32, "rbtree cap")
	rootCmd.PersistentFlags().StringVar(&SwitchTable, "switch", "", "switch table time interval") // day week month // 默认按一天分
	rootCmd.PersistentFlags().Int64Var(&optimizeFlag, "optimizeFlag", defaultOptimizeFlag, "optimize flag")
	// Bind flags to Viper
	err := viper.BindPFlag("save", rootCmd.PersistentFlags().Lookup("save"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("src", rootCmd.PersistentFlags().Lookup("src"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("dst", rootCmd.PersistentFlags().Lookup("dst"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("fileName", rootCmd.PersistentFlags().Lookup("fileName"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("pkgName", rootCmd.PersistentFlags().Lookup("pkgName"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("unloadKey", rootCmd.PersistentFlags().Lookup("unloadKey"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("unload", rootCmd.PersistentFlags().Lookup("unload"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("warningFlag", rootCmd.PersistentFlags().Lookup("warningFlag"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("maxInsertRows", rootCmd.PersistentFlags().Lookup("maxInsertRows"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("queueThreshold", rootCmd.PersistentFlags().Lookup("queueThreshold"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("queueEmptySleepTime", rootCmd.PersistentFlags().Lookup("queueEmptySleepTime"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("separate", rootCmd.PersistentFlags().Lookup("separate"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("rbtreeCap", rootCmd.PersistentFlags().Lookup("rbtreeCap"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("switch", rootCmd.PersistentFlags().Lookup("switch"))
	if err != nil {
		return err
	}
	err = viper.BindPFlag("optimizeFlag", rootCmd.PersistentFlags().Lookup("optimizeFlag"))
	if err != nil {
		return err
	}

	return nil
}

func Run(_ *cobra.Command, args []string) {
	var err error
	var persistPkgName, persistPkgPath string
	var pwd string

	// 初始化参数
	pwd, err = os.Getwd()
	if err != nil {
		fmt.Printf("Could not GetWd(): %s (skip)\n", err)
	}

	if srcDir == "" {
		srcDir = pwd
	} else {
		persistPkgPath = srcDir
		persistPkgName = persistPkgPath[strings.LastIndex(persistPkgPath, "/")+1:]
		// srcDir = path.Join(goPath, "src", srcDir)
	}
	if dstDir == "" {
		dstDir = pwd
	} else {
		// dstDir = path.Join(goPath, "src", dstDir)
	}

	if SwitchTable != "day" && SwitchTable != "week" && SwitchTable != "month" {
		SwitchTable = ""
	} else {
		if unload {
			panic("switch table can not unload")
		}
	}

	// 数据库冲突生成地址
	bombDir := "./" + strings.Replace(strings.Replace(strings.Replace(dstDir, "\\", "_", -1), ":", "", -1), "/", "_", -1)

	srcBegin := findFirstFromRight(bombDir, "src")
	if srcBegin == -1 {
		srcBegin = 0
	}
	bombDir = "_" + bombDir[srcBegin:]

	argsInfo := config.ArgsInfo{
		Save:                save,
		PersistPkgPath:      persistPkgPath,
		PersistPkgName:      persistPkgName,
		UnloadKey:           unloadKey,
		Unload:              unload,
		WarningFlag:         warningFlag,
		BombDir:             bombDir,
		OptimizeFlag:        optimizeFlag,
		MaxInsertRows:       maxInsertRows,
		QueueThreshold:      queueThreshold,
		QueueEmptySleepTime: queueEmptySleepTime,
		FileName:            fileName,
		NeedSeparate:        NeedChangeDataBase,
		SwitchTable:         SwitchTable,
		RBTreeCap:           int32(rbtreeCap),
	}

	wg := &sync.WaitGroup{}

	for _, tableName := range args {
		wg.Add(1)
		go core.GenPersistMethod(wg, srcDir, fileName, dstDir, pkgName, tableName, &argsInfo)
	}

	wg.Wait()

}

func Execute() error {
	rootCmd.Use = "persist"
	rootCmd.Short = "A simple and easy-to-use data persistence solution."
	rootCmd.Run = Run
	return rootCmd.Execute()
}

func main() {
	err := Execute()
	if err != nil {
		panic(err)
	}
}
