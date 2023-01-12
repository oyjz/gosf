package gosf

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// IsRelease 判断是否是二进制运行
func IsRelease() bool {
	arg1 := strings.ToLower(os.Args[0])
	isRelease := false
	if strings.Index(arg1, "go-build") < 0 {
		isRelease = true
	}
	return isRelease
}

// RootPath 获取运行目录
func RootPath() string {
	var rootPath string
	var err error
	// 设置程序根目录
	rootPath, err = os.Getwd()
	if err != nil {
		return "./"
	}
	if IsRelease() {
		// 如果是二进制运行，是取对应二进制文件所在目录
		var ex string
		ex, err = os.Executable()
		if err != nil {
			return "./"
		}
		rootPath = filepath.Dir(ex)
	}
	return rootPath
}

// PathExists 判断文件/文件夹是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// func MkdirAll(path string) {
// 	res, _ := PathExists(path)
// 	if !res {
// 		_ = os.MkdirAll(path, os.ModePerm)
// 	}
// }

// Exit 退出程序
func Exit(v ...any) {
	if len(v) > 0 {
		fmt.Println(v)
		log.Println(v)
	}
	os.Exit(1)
}
