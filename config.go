package gosf

import (
	"gosf/config"
	"os"
)

// Configer 获取配置实例
func Configer(file string) config.Config {
	checkPath, err := PathExists(file)
	if !checkPath {
		Exit("config file not found", err)
	}
	f, err := os.Open(file)
	if err != nil {
		Exit("config file open found", err)
	}
	defer f.Close()

	config, err := config.FromJson(f)
	if err != nil {
		Exit("config file parse found", err)
	}

	return config
}
