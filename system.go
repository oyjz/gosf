package gosf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// FilesystemInfo 磁盘挂载信息
type FilesystemInfo struct {
	Filesystem string `json:"filesystem"`
	Size       string `json:"size"`
	Used       string `json:"used"`
	Available  string `json:"available"`
	Use        string `json:"use"`
	Mount      string `json:"mount"`
}

type System struct {
	Cpu        string   `json:"cpu"`
	InternalIp []string `json:"internal_ip"`
	PublicIp   string   `json:"public_ip"`
	Disk       string   `json:"disk"`
	Mac        string   `json:"mac"`
	Now        string   `json:"now"`
}

func NewSystem() *System {
	now := time.Now().Format("2006-01-02 15:04:05")

	return &System{
		Now: now,
	}
}

func (p *System) GetCpu() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var modelName string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			fields := strings.Fields(line)
			if len(fields) > 2 {
				modelName = fields[2]
				break // 假设每个CPU的model name都是一样的，所以找到第一个就退出循环
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	p.Cpu = modelName

	return p.Cpu
}

func (p *System) GetDisk() string {
	cmd := exec.Command("df")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("df failed:", err)
		return ""
	}

	lines := strings.Split(string(out), "\n")
	var buffer bytes.Buffer
	buffer.WriteString("[")
	for i, line := range lines {
		if i == 0 {
			continue // 跳过第一行
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		if i > 1 {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf("{\"filesystem\":\"%s\",\"size\":\"%s\",\"used\":\"%s\",\"available\":\"%s\",\"use\":\"%s\",\"mount\":\"%s\"}", fields[0], fields[1], fields[2], fields[3], fields[4], fields[5]))
	}
	buffer.WriteString("]")

	disk := buffer.String()
	p.Disk = disk
	return p.Disk
}

func (p *System) GetIp() []string {
	var ips []string
	// 构造命令对象
	cmd := exec.Command("ip", "addr", "show")

	// 执行命令并获取输出
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("ip get failed:", err)
		return ips
	}

	// 将命令输出按行分割
	lines := strings.Split(string(output), "\n")

	// 遍历每一行输出
	for _, line := range lines {
		// 过滤包含 "inet " 的行且不包含 "127.0.0.1" 的行
		if strings.Contains(line, "inet ") && !strings.Contains(line, "127.0.0.1") {
			// 使用空格分割并取第二个字段，即 IP 地址
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ips = append(ips, fields[1])
			}
		}
	}

	p.InternalIp = ips

	return p.InternalIp
}

func (p *System) GetPublicIP() string {
	resp, err := http.Get("http://ifconfig.me")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	p.PublicIp = string(body)
	return p.PublicIp
}

// isPrivateSubnetCIDR 检查给定的IP是否属于私有网络CIDR
func (p *System) isPrivateSubnetCIDR(ip net.IP) bool {
	privateCIDRs := []string{
		"10.0.0.0/8",     // 私有网络1
		"172.16.0.0/12",  // 私有网络2
		"192.168.0.0/16", // 私有网络3
	}

	for _, cidr := range privateCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			fmt.Printf("parse error on CIDR %s: %s\n", cidr, err)
			continue
		}
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// GetInternalIPs 获取所有内网IP地址
func (p *System) GetInternalIPs() []string {
	var internalIPs []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	for _, iface := range interfaces {
		// 忽略down状态的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// 忽略回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4() // 只处理IPv4地址
			if ip == nil {
				continue // 忽略IPv6地址
			}

			// 检查IP是否属于私有网络
			if p.isPrivateSubnetCIDR(ip) {
				internalIPs = append(internalIPs, ip.String())
			}
		}
	}

	return internalIPs
}

func (p *System) GetMac() string {
	// 构造命令对象
	cmd := exec.Command("ip", "addr", "show")

	// 执行命令并获取输出
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("mac get failed", err)
		return ""
	}

	// 将命令输出按行分割
	lines := strings.Split(string(output), "\n")
	var macs []string

	// 遍历每一行输出
	for _, line := range lines {
		// 过滤包含 "link/ether" 的行
		if strings.Contains(line, "link/ether") {
			// 使用空格分割并取第二个字段，即 MAC 地址
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				macs = append(macs, fields[1])
			}
		}
	}

	p.Mac = strings.Join(macs, ",")
	// 输出结果
	return p.Mac
}

// GetAvailable 获取指定目录可用磁盘大小
func (p *System) GetAvailable(path string) int {
	disk := p.GetDisk()
	// 解析磁盘数据
	var filesystems []FilesystemInfo
	if err := json.Unmarshal([]byte(disk), &filesystems); err != nil {
		fmt.Println("Error parsing disk JSON:", err)
	}

	// 遍历文件系统信息，判断指定分区剩余空间是否超过 10M
	available := ""
	for _, fs := range filesystems {
		if strings.HasPrefix(path, fs.Mount) {
			available = fs.Available
		}
	}

	ava, _ := strconv.Atoi(available)
	return ava
}
