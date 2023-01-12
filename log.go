package gosf

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Logger struct {
	// *log.Logger
	Path        string
	MaxSize     int
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	FatalLogger *log.Logger
}

func (app *Gosf) Log() *Logger {
	var debugLogger *log.Logger
	var infoLogger *log.Logger
	var errorLogger *log.Logger
	var fatalLogger *log.Logger
	if app.IsRelease {
		debugLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)
		infoLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)
		errorLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)
		fatalLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)
	} else {
		debugLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
		infoLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
		errorLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
		fatalLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
	}
	if len(app.Config.Name) > 0 {
		debugLogger.SetPrefix("[" + app.Config.Name + "] ")
		infoLogger.SetPrefix("[" + app.Config.Name + "] ")
		errorLogger.SetPrefix("[" + app.Config.Name + "] ")
		fatalLogger.SetPrefix("[" + app.Config.Name + "] ")
	}
	if len(app.Config.LogPath) > 0 {
		checkPath, _ := PathExists(app.Config.LogPath)
		if !checkPath {
			_ = os.MkdirAll(app.Config.LogPath, os.ModePerm)
		}
		app.Config.LogPath = strings.TrimRight(app.Config.LogPath, "\\")
		app.Config.LogPath = strings.TrimRight(app.Config.LogPath, "/")
	}

	// 默认最大10M
	logMaxSize := 5
	if app.Config.LogMaxSize > 0 {
		logMaxSize = app.Config.LogMaxSize
	}

	var loggerMain = new(Logger)
	loggerMain.MaxSize = logMaxSize
	loggerMain.Path = app.Config.LogPath
	loggerMain.DebugLogger = debugLogger
	loggerMain.InfoLogger = infoLogger
	loggerMain.ErrorLogger = errorLogger
	loggerMain.FatalLogger = fatalLogger

	return loggerMain
}

func (logger *Logger) Debug(v ...any) {
	if len(logger.Path) > 0 {
		logger.DebugLogger.SetOutput(logger.GetLogFile("debug"))
	}
	logger.DebugLogger.Println(v)
}
func (logger *Logger) Info(v ...any) {
	if len(logger.Path) > 0 {
		logger.InfoLogger.SetOutput(logger.GetLogFile("info"))
	}
	logger.InfoLogger.Println(v)
}
func (logger *Logger) Error(v ...any) {
	if len(logger.Path) > 0 {
		logger.ErrorLogger.SetOutput(logger.GetLogFile("error"))
	}
	logger.ErrorLogger.Println(v)
}
func (logger *Logger) Fatal(v ...any) {
	if len(logger.Path) > 0 {
		logger.FatalLogger.SetOutput(logger.GetLogFile("fatal"))
	}
	logger.FatalLogger.Fatalln(v)
}

func (logger *Logger) GetLogFile(level string) *os.File {
	if len(logger.Path) > 0 {
		path := logger.Path
		level = level + "\\"
		// 设置时区
		var cstSh, _ = time.LoadLocation("Asia/Shanghai") // 上海
		t := time.Now().In(cstSh)
		// 日志目录，不存在则创建
		logPath := fmt.Sprintf("%s\\%s", path, level)
		checkPath, _ := PathExists(logPath)
		if !checkPath {
			_ = os.MkdirAll(logPath, os.ModePerm)
		}
		// 日志文件
		logFile := fmt.Sprintf("%s%s.log", logPath, t.Format("20060102"))
		// 判断文件大小
		logFileExist, _ := PathExists(logFile)
		if logFileExist {
			fs, err := os.Stat(logFile)
			if err != nil {
				fmt.Println("log file stat failed", err)
				os.Exit(1)
			}
			fileSize := int(fs.Size() / 1024 / 1024)
			if fileSize >= logger.MaxSize {
				// 日志切割
				err1 := os.Rename(logFile, fmt.Sprintf("%s-%s", logFile, t.Format("150405")))
				if err1 != nil {
					fmt.Println("log file rename failed", err1)
					os.Exit(1)
				}
			}
		}
		file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
		if err != nil {
			fmt.Println("log file open failed", err)
			os.Exit(1)
		}

		return file
	} else {
		return nil
	}
}
