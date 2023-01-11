package gosf

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func (app *Gosf) Log() *log.Logger {
	var logger *log.Logger
	if app.IsRelease {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
	}
	if len(app.Config.Name) > 0 {
		logger.SetPrefix("[" + app.Config.Name + "] ")
	}
	if len(app.Config.LogFile) > 0 {
		filePath, _ := filepath.Split(app.Config.LogFile)
		checkPath, _ := PathExists(filePath)
		if !checkPath {
			_ = os.MkdirAll(filePath, os.ModePerm)
		}
		logFile, err := os.OpenFile(app.Config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
		if err != nil {
			fmt.Println("log file open failed", err)
			os.Exit(1)
		}
		logger.SetOutput(logFile)
	}
	return logger
}
