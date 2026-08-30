package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cn-joyconn/goutils/logger"
)

// installBinaryBackup 将当前二进制复制为 <binary>.bak，
// 供 systemd ExecStartPre 在二进制损坏(如掉电导致文件系统损坏)时自动恢复。
func InstallBinaryBackup(ilog logger.ILogger) {
	exePath, err := os.Executable()
	if err != nil {
		ilog.Warnf("获取可执行文件路径失败, 跳过二进制备份: %v", err)
		return
	}
	src, err := os.Open(exePath)
	if err != nil {
		ilog.Warnf("打开二进制失败, 跳过备份: %v", err)
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(exePath+".bak", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		ilog.Warnf("创建备份文件失败, 跳过备份: %v", err)
		return
	}
	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		ilog.Warnf("写入备份文件失败: %v", err)
		return
	}
	if err = dst.Close(); err != nil {
		ilog.Warnf("关闭备份文件失败: %v", err)
		return
	}
	dst.Sync()
	fmt.Println("已创建二进制备份: " + exePath + ".bak")
}

// patchSystemdSelfHeal 在 systemd 单元文件中注入 ExecStartPre：
// 服务启动前校验二进制 ELF 头(7f 45 4c 46)，若文件损坏且存在 .bak 则自动恢复后再启动。
// 仅 linux(systemd) 生效；kardianos/service v1.2.2 模板不支持 Option 注入 ExecStartPre，故做后处理。
func PatchSystemdSelfHeal(ilog logger.ILogger, svrName string) {
	if runtime.GOOS != "linux" {
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		ilog.Warnf("获取可执行文件路径失败, 跳过 systemd 自愈配置: %v", err)
		return
	}
	unitPath := "/etc/systemd/system/" + svrName + ".service"
	unit, err := os.ReadFile(unitPath)
	if err != nil {
		ilog.Warnf("读取 %s 失败, 跳过 systemd 自愈配置: %v", unitPath, err)
		return
	}
	content := string(unit)
	if strings.Contains(content, "ExecStartPre=") {
		return // 已注入，避免重复
	}
	// ELF 魔数 7f 45 4c 46；损坏且存在备份时恢复
	guard := fmt.Sprintf("ExecStartPre=/bin/sh -c 'if ! head -c 4 \"%s\" | od -An -tx1 | grep -q \"7f 45 4c 46\"; then if [ -f \"%s.bak\" ]; then cp -f \"%s.bak\" \"%s\"; echo \"binary corrupted, restored from backup\"; fi; fi'",
		exePath, exePath, exePath, exePath)
	content = strings.Replace(content, "ExecStart=", guard+"\nExecStart=", 1)
	if err = os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		ilog.Warnf("写入 %s 失败, systemd 自愈配置未生效: %v", unitPath, err)
		return
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		ilog.Warnf("systemctl daemon-reload 失败: %v, output: %s", err, string(out))
	}
}
