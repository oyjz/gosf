package gosf

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	// if os.IsNotExist(err) {
	// 	return false, nil
	// }
	return false, err
}

func MkdirAll(path string) {
	res, _ := PathExists(path)
	if !res {
		_ = os.MkdirAll(path, os.ModePerm)
	}
}

// Exit 退出程序
func Exit(v ...any) {
	if len(v) > 0 {
		log.Println(v)
	}
	os.Exit(1)
}

func IsProcessRunning(processName string) (bool, error) {
	files, err := os.ReadDir("/proc")
	if err != nil {
		return false, err
	}

	for _, file := range files {
		if file.IsDir() {
			pid := file.Name()
			// 检查是否为数字，避免误判
			if _, err := os.Stat("/proc/" + pid + "/cmdline"); err == nil {
				cmdline, err := os.ReadFile("/proc/" + pid + "/cmdline")
				if err != nil {
					return false, err
				}
				if strings.Contains(string(cmdline), processName) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func StartProgram(programDir, programName string, args ...string) error {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 切换到程序主目录
	err = os.Chdir(programDir)
	if err != nil {
		return err
	}
	defer os.Chdir(wd) // 切换回原工作目录

	// 执行程序
	var cmd *exec.Cmd
	if len(args) > 0 {
		cmd = exec.Command(programName, args...)
	} else {
		cmd = exec.Command(programName)
	}
	cmd.Stdout = os.Stdout // 将程序的标准输出连接到主程序的标准输出
	cmd.Stderr = os.Stderr // 将程序的标准错误连接到主程序的标准错误
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func InForString(items []string, item string) bool {
	sort.Strings(items)
	index := sort.SearchStrings(items, item)
	// index的取值：[0,len(str_array)]
	if index < len(items) && items[index] == item {
		// 需要注意此处的判断，先判断 &&左侧的条件，如果不满足则结束此处判断，不会再进行右侧的判断
		return true
	}
	return false
}

func ReadFile(filePath string) string {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	return string(content)
}

// 清空文件内容
func ClearFileContent(filePath string) error {
	// 打开文件以进行写入（截断模式）
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// 清空文件内容
	if _, err := file.WriteString(""); err != nil {
		return err
	}

	return nil
}

// 替换文件
func ReplaceFile(sourcePath, destPath string) error {
	// 打开源文件
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	_ = os.Remove(destPath)

	// 创建或打开目标文件
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 复制源文件内容到目标文件
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 关闭目标文件，确保写入完成
	err = destFile.Close()
	if err != nil {
		return err
	}

	// 删除源文件
	err = os.Remove(sourcePath)
	if err != nil {
		return err
	}

	return nil
}

func KillProcessByName(processName string) error {
	// 执行ps命令获取进程列表
	psCmd := exec.Command("ps", "-A", "-o", "pid,cmd")
	output, err := psCmd.Output()
	if err != nil {
		return err
	}

	// 解析ps命令输出，查找目标进程并杀死
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.Contains(fields[1], processName) {
			pid := fields[0]
			killCmd := exec.Command("kill", "-9", pid)
			if err := killCmd.Run(); err != nil {
				return err
			}
			fmt.Printf("Process with name '%s' (PID: %s) killed successfully.\n", processName, pid)
		}
	}

	return nil
}

func GetFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)
	md5Str := hex.EncodeToString(hashInBytes)
	return md5Str, nil
}
