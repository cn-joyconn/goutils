package services

import (
	"crypto/md5"
	"encoding/hex"
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
	if err = copyBinary(exePath, exePath+".bak"); err != nil {
		ilog.Warnf("创建二进制备份失败: %v", err)
		return
	}
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
	if strings.Contains(content, "restored from backup") {
		return // 已注入，避免重复
	}
	// "$0" 传参传递二进制路径, 避免路径含空格/特殊字符的转义问题
	guard := fmt.Sprintf(`ExecStartPre=/bin/sh -c 'if ! "$0" -s runtest | grep -q success; then if [ -f "$0.bak" ]; then cp -f "$0.bak" "$0"; echo "binary corrupted, restored from backup"; fi; fi' "%s"`, exePath)
	content = strings.Replace(content, "ExecStart=", guard+"\nExecStart=", 1)
	if err = os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		ilog.Warnf("写入 %s 失败, systemd 自愈配置未生效: %v", unitPath, err)
		return
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		ilog.Warnf("systemctl daemon-reload 失败: %v, output: %s", err, string(out))
	}
}

// refreshBinaryBackup 每次启动时比对当前二进制与 .bak 的 md5，
// 不一致(或 .bak 不存在)时将当前二进制刷新为新的 .bak。
// 说明：二进制无法启动时本逻辑不会执行，
// "无法启动"场景由 systemd ExecStartPre(-runtest 自检)从 .bak 恢复兜底。
func RefreshBinaryBackup(ilog logger.ILogger) {
	exePath, err := os.Executable()
	if err != nil {
		ilog.Warnf("获取可执行文件路径失败, 跳过备份刷新: %v", err)
		return
	}
	curMD5, err := fileMD5(exePath)
	if err != nil {
		ilog.Warnf("计算当前二进制 md5 失败, 跳过备份刷新: %v", err)
		return
	}
	bakPath := exePath + ".bak"
	if bakMD5, err := fileMD5(bakPath); err == nil && bakMD5 == curMD5 {
		return // 与备份一致，无需刷新
	}
	if err = copyBinary(exePath, bakPath); err != nil {
		ilog.Warnf("刷新二进制备份失败: %v", err)
		return
	}
	ilog.Infof("二进制与备份不一致, 已刷新备份: %s", bakPath)
}

// copyBinary 以 0755 权限将 src 复制为 dst，写完后落盘(sync)再关闭。
// 说明：os.OpenFile 的权限参数仅在文件创建时生效且受 umask 影响，
// 文件已存在时保留原权限，故复制完成后显式 Chmod 0755 确保可执行。
func copyBinary(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	if err = dst.Sync(); err != nil {
		dst.Close()
		return err
	}
	if err = dst.Close(); err != nil {
		return err
	}
	return os.Chmod(dstPath, 0755)
}

// fileMD5 计算文件 md5 的十六进制字符串。
func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
