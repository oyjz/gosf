package gosf

import (
	"os"

	"github.com/oyjz/gosf/config"
)

// Configer 获取配置实例
func Configer(file string) config.Config {
	checkPath, err := PathExists(file)
	if !checkPath || err != nil {
		Exit(err, "config file not found")
	}
	f, err := os.Open(file)
	PanicErr(err, "config file open error")
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)

	value, err := config.FromJson(f)
	PanicErr(err, "config file parse error")

	return value
}
