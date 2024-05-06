package gosf

import (
	"fmt"
	"os"
	"sync"
)

type Gosf struct {
	Config    Config
	IsRelease bool
	Logger    *Logger
	WaitGroup sync.WaitGroup
	Task      []func(app *Gosf)
}

// Config APP配置
type Config struct {
	Name       string // APP 名称
	LogPath    string // APP日志路径
	ConfigFile string // APP日志路径
	LogMaxSize int    // 日志文件最大大小，单位M
}

// App 获取一个新的APP实例
// Deprecated: User gosf.NewApp()
func App(config Config) Gosf {
	var app Gosf
	var task []func(app *Gosf)
	app.Config = config
	app.IsRelease = IsRelease()
	app.WaitGroup = sync.WaitGroup{}
	app.Logger = app.Log()
	app.Task = task
	// 设置日志`
	return app
}

// NewApp 获取一个新的APP实例 返回Gosf指针
func NewApp(config Config) *Gosf {
	app := &Gosf{
		Config:    config,
		IsRelease: IsRelease(),
		WaitGroup: sync.WaitGroup{},
		// Task 初始化时可能为空，但通常会在其他地方填充
		Task: []func(*Gosf){}, // 初始化一个空的函数切片
	}

	app.Logger = app.Log()
	return app
}

// Run 执行任务
func (app *Gosf) Run() {

	// 创建一个Daemon对象
	// logFile := base.RootPath + "\\storage\\log\\daemon.log"
	// d := godaemon.NewDaemon("")
	// // 调整一些运行参数(可选)
	// d.MaxCount = 30 // 最大重启次数
	// // 执行守护进程模式
	// d.Run()

	app.WaitGroup.Add(len(app.Task))

	for i, n := 0, len(app.Task); i < n; i++ {
		go func(f func(app *Gosf), app *Gosf) {
			defer app.WaitGroup.Done()
			f(app)
		}(app.Task[i], app)
	}

	app.WaitGroup.Wait()
}

// Add 添加任务
func (app *Gosf) Add(f func(app *Gosf)) {
	app.Task = append(app.Task, f)
}

// PanicErr 错误处理
func (app *Gosf) PanicErr(err error, v ...any) {
	if err != nil {
		app.Logger.Fatal(v, err)
		if app.IsRelease {
			fmt.Println(v, err.Error())
			os.Exit(3)
		} else {
			panic(err)
		}
	}
}

// FmtLog 终端输出，日志也记录
func (app *Gosf) FmtLog(v ...any) {
	fmt.Println(v)
	app.Logger.Info(v)
}

// Exit 中断程序
func (app *Gosf) Exit(v ...any) {
	if len(v) > 0 {
		fmt.Println(v)
		app.Logger.Fatal(v)
	}
	os.Exit(1)
}
