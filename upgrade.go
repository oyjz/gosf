package gosf

import (
	"archive/tar"
	"compress/gzip"
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
	p.tmpName = p.AppName + ".tar.gz"
	p.tmpPath = filepath.Join(tempDir, p.tmpName)
	p.bakName = fmt.Sprintf("%s-%s.tar.gz", p.AppName, time.Now().Format("20060102150405"))
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
		if err := p.backup(); err != nil {
			fmt.Println("backup failed:", err)
			return
		}

		// 安装更新
		if err := p.install(); err != nil {
			_ = os.Remove(p.bakPath)
			_ = os.Remove(p.tmpName)
			fmt.Println("install failed:", err)
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
func (p *Upgrade) backup() error {

	// 创建压缩文件
	backupFile, err := os.Create(p.bakPath)
	if err != nil {
		fmt.Printf("failed to create backup file: %v\n", err)
		return err
	}
	defer backupFile.Close()

	// 创建 gzip.Writer
	gzipWriter := gzip.NewWriter(backupFile)
	defer gzipWriter.Close()

	// 创建 tar.Writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// 遍历源目录下的文件和子目录
	err = filepath.Walk(p.AppPath, func(path string, info os.FileInfo, err error) error {
		// 排除 logs 目录和 app.log 文件
		if info.IsDir() && info.Name() == "logs" {
			return filepath.SkipDir
		}
		// 跳过扩展名为 .log 的文件
		if !info.IsDir() && strings.ToLower(filepath.Ext(info.Name())) == ".log" {
			return nil
		}
		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("Failed to open file %s: %v\n", path, err)
			return nil
		}
		defer file.Close()

		// 获取相对路径
		relPath, err := filepath.Rel(p.AppPath, path)
		if err != nil {
			fmt.Printf("Failed to get relative path for file %s: %v\n", path, err)
			return nil
		}

		// 创建 tar 文件头
		header := &tar.Header{
			Name:    relPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		// 写入 tar 文件头
		if err := tarWriter.WriteHeader(header); err != nil {
			fmt.Printf("Failed to write tar header for file %s: %v\n", path, err)
			return nil
		}

		// 写入文件内容
		if _, err := io.Copy(tarWriter, file); err != nil {
			fmt.Printf("Failed to write file %s to tar: %v\n", path, err)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Failed to walk directory: %v\n", err)
		return err
	}

	fmt.Println("backup ok:", p.bakPath)
	return nil
}

// 安装文件
func (p *Upgrade) install() error {

	// 下载升级包
	resp, err := http.Get(p.FileUrl)
	if err != nil {
		fmt.Println("download failed", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("frp get http status error", resp.Status)
		return fmt.Errorf("failed to download FRP: %s", resp.Status)
	}

	// 创建目标文件
	targetFile, err := os.Create(p.tmpPath)
	if err != nil {
		fmt.Println("tmp gz file create failed", err)
		return err
	}
	defer targetFile.Close()

	// 将下载的内容保存到目标文件
	_, err = io.Copy(targetFile, resp.Body)
	if err != nil {
		fmt.Println("tmp gz file save failed", err)
		return err
	}

	// 重置文件指针为开头
	_, err = targetFile.Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println("failed to reset tmp gz file pointer:", err)
		return err
	}

	// 创建gzip reader
	gzipReader, err := gzip.NewReader(targetFile)
	if err != nil {
		fmt.Println("failed to create gzip reader:", err)
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("tmp gz file reader next failed", err)
			return err
		}

		subFile := filepath.Join(p.appPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(subFile, 0755); err != nil {
				fmt.Println(" tmp gz file reader failed 1", err)
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(subFile)
			if err != nil {
				fmt.Println("tmp gz file reader failed 2", err)
				return err
			}
			defer file.Close()
			if _, err := io.Copy(file, tarReader); err != nil {
				fmt.Println("tmp gz file reader failed 3", err)
				return err
			}
		default:
			fmt.Println("frp unknown file type", header.Typeflag)
			return fmt.Errorf("unknown file type %s in tar", header.Typeflag)
		}
	}

	// 删除压缩包
	_ = os.Remove(p.tmpPath)

	return nil
}

// 重启应用
func (p *Upgrade) restartApp() error {

	err := StartProgram(p.AppPath, "./"+p.AppName)

	if err != nil {
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
