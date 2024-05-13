package gosf

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Upgrade struct {
	ID       string // 【需要设置】当前服务器标识（根据各自业务而定，可以是自定义的唯一标识，也可以是公网IP、MAC地址等）
	AppPath  string // 【必须设置】APP 所在路劲
	AppName  string // 【必须设置】APP 名称
	CheckUrl string // 【必须设置】最新版本文件MD5
	FileUrl  string // 【必须设置】最新版本文件下载链接
	newMd5   string // 最新版本文件MD5
	appPath  string // APP文件全路径
	tmpName  string // APP临时文件名称
	tmpPath  string // APP临时文件全路径
	bakName  string // APP备份文件名称
	bakPath  string // APP备份文件全路径
}

func NewUpgrade() *Upgrade {
	return &Upgrade{}
}

func (p *Upgrade) setConfig() {
	// 设置标识默认为公网IP
	if p.ID == "" {
		p.ID = NewSystem().GetPublicIP()
	}

	// 获取系统临时目录路径
	tempDir := os.TempDir()
	p.appPath = filepath.Join(p.AppPath, p.AppName)
	p.tmpName = p.AppName + "_tmp"
	p.tmpPath = filepath.Join(p.AppPath, p.tmpName)
	p.bakName = p.AppName + "_bak"
	p.bakPath = filepath.Join(tempDir, p.bakName)
}

// Run 定时检测升级
func (p *Upgrade) Run() {
	p.Do()

	for {
		rand.Seed(time.Now().UnixNano())
		randomInt := 240 + rand.Intn(120)

		time.Sleep(time.Duration(randomInt) * time.Minute)

		p.Do()
	}
}

// Do 检测升级
func (p *Upgrade) Do() {
	now := time.Now().Format("2006-01-02 15:04:05")

	// 初始化配置
	p.setConfig()

	// 检验是否完成相关配置
	if p.CheckUrl == "" || p.FileUrl == "" || p.AppName == "" || p.AppPath == "" {
		fmt.Println(now, "incomplete upgrade configuration")
		return
	}

	// 检查更新
	if p.updateAvailable() {

		// 判断剩余磁盘空间是否够，足够才下载
		appFileInfo, err := os.Stat(p.appPath)
		var appFileSize int64
		if err != nil {
			fmt.Println("app file stat error", err)
			appFileSize = 10240
		} else {
			appFileSize = appFileInfo.Size()/1024 + 2048
		}

		available := NewSystem().GetAvailable(p.appPath)
		if available < int(appFileSize) {
			fmt.Println("The available space cannot be less than 8M")
			return
		}

		fmt.Println(now, "start upgrade...")
		// 备份原程序
		if err := p.backupCurrentVersion(p.appPath); err != nil {
			fmt.Println("backup app failed:", err)
			return
		}

		// 下载更新
		if err := p.downloadUpdate(); err != nil {
			_ = os.Remove(p.bakPath)
			_ = os.Remove(p.tmpPath)
			fmt.Println("download update failed:", err)
			return
		}

		// 安装更新
		if err := p.installUpdate(); err != nil {
			return
		}

		_ = os.Remove(p.bakPath)

		// 重启程序
		if err := p.restartApp(); err != nil {
			fmt.Println("restart failed:", err)
			return
		}
		fmt.Println(now, "upgrade success")
	} else {
		// fmt.Println(now, "guard is new")
	}
}

// 校验是否需要升级，接口返回最新升级文件MD5，通过校验本地文件MD5是否一致来判断是否需要升级
func (p *Upgrade) updateAvailable() bool {
	fileMd5, err := GetFileMD5(p.appPath)
	if err != nil {
		fmt.Println("md5 get failed", err)
		return false
	}

	err, body := NewRequest().
		Url(p.CheckUrl).
		Timeout(10).
		AddParam("id", p.ID).
		AddHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36").
		Get()
	if err != nil {
		fmt.Println("md5 request failed", err)
		return false
	}

	// 校验返回是否是32位MD5格式字符串
	reg := regexp.MustCompile("^[a-z0-9]{32}$")

	// 如果不是MD5字符串 或者 MD5字符串和当前文件MD5相同，则无需升级
	if !reg.MatchString(string(body)) || string(body) == fileMd5 {
		return false
	}

	// 设置最新升级文件MD5
	p.newMd5 = string(body)

	return true
}

// 备份当前文件
func (p *Upgrade) backupCurrentVersion(originalPath string) error {
	// 打开原始文件
	originalFile, err := os.Open(originalPath)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	// 创建备份文件
	backupFile, err := os.Create(p.bakPath)
	if err != nil {
		return err
	}
	defer backupFile.Close()

	// 复制原始文件内容到备份文件
	_, err = io.Copy(backupFile, originalFile)
	if err != nil {
		_ = os.Remove(p.bakPath)
		return err
	}

	fmt.Println("backup ok:", p.bakPath)
	return nil
}

// 下载升级文件
func (p *Upgrade) downloadUpdate() error {
	resp, err := http.Get(p.FileUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(p.tmpPath)
	if err != nil {
		fmt.Println("tmp file create failed", err)
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("tmp file save failed", err)
		_ = os.Remove(p.tmpPath)
		return err
	}
	return nil
}

// 安装文件
func (p *Upgrade) installUpdate() error {
	fileMd5, err := GetFileMD5(p.tmpPath)
	if err != nil {
		fmt.Println("md5 get failed", err)
		_ = os.Remove(p.tmpPath)
		_ = os.Remove(p.bakPath)
		return err
	}
	if fileMd5 == p.newMd5 {
		// 实现安装更新的逻辑，可以直接替换程序文件
		err = os.Rename(p.tmpPath, p.appPath)
		if err != nil {
			fmt.Println("tmp to app failed:", err)
			return err
		}
		// 修改文件权限
		if err = os.Chmod(p.appPath, 0777); err != nil {
			fmt.Println("Failed to change app permission:", err)
			return err
		}
		return nil
	} else {
		_ = os.Remove(p.tmpPath)
		_ = os.Remove(p.bakPath)
		fmt.Println("download file md5 is error", fileMd5)
		return errors.New("download file md5 is error")
	}
}

// 重启应用
func (p *Upgrade) restartApp() error {

	err := StartProgram(p.AppPath, "./"+p.AppName)

	if err != nil {
		_ = os.Remove(p.bakPath)
		return err
	}
	os.Exit(1)

	return nil
}

// KillApp 杀掉原进程【程序启动的时候用来检测是否有残留的进程，建议放init方法里】
func (p *Upgrade) KillApp(app *Gosf) error {
	p.AppName = app.Config.Name
	p.AppPath = app.Config.Path
	// 初始化配置
	p.setConfig()

	if p.AppName == "" {
		fmt.Println("AppName is empty")
		return nil
	}

	// 获取所有进程
	var cmd *exec.Cmd
	cmd = exec.Command("ps", "-A", "-o", "pid,etime,comm")

	ppid := os.Getppid()
	pid := os.Getpid()

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// 检查输出中的PID并关闭
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pidStr := fields[0]
			etime := fields[1]

			seconds, err := p.timeToSeconds(etime)
			if err != nil {
				fmt.Println("get etime failed:", err)
				continue
			}

			pidInt, _ := strconv.Atoi(pidStr)

			// 如果运行时间超过10s且是目标应用程序，则杀死进程
			if seconds > 60 && strings.Contains(fields[2], p.AppName) && pidInt != ppid && pidInt != pid {
				killCmd := exec.Command("kill", pidStr)
				_ = killCmd.Run()
				fmt.Println("Killed process with PID:", pidStr)
			}
		}
	}
	return nil
}

// 读取到的进程运行时间转成秒
func (p *Upgrade) timeToSeconds(timeStr string) (int, error) {
	// 按冒号分割小时和分钟
	timeParts := strings.Split(timeStr, ":")
	if len(timeParts) != 2 {
		timeParts = strings.Split(timeStr, "h")
		if len(timeParts) != 2 {
			return 0, fmt.Errorf("invalid time format: %s", timeStr)
		}
	}

	// 解析小时和分钟
	minute, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour: %s", timeParts[0])
	}
	second, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute: %s", timeParts[1])
	}

	// 将小时和分钟转换为秒数
	minuteSeconds := minute * 60

	// 总秒数
	totalSeconds := second + minuteSeconds

	return totalSeconds, nil
}
