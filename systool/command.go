package systool

import (
	"bytes"
	"os/exec"
	"time"
	"log"
	"github.com/cn-joyconn/go-joytools/strtool"
)

func CmdOut(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func CmdOutBytes(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}

func CmdOutNoLn(name string, arg ...string) (out string, err error) {
	out, err = CmdOut(name, arg...)
	if err != nil {
		return
	}
	return strtool.TrimRightSpace(string(out)), nil
}

func CmdRunWithTimeout(cmd *exec.Cmd, timeout time.Duration) (error, bool) {
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	select {
	case <-time.After(timeout):
		//timeout
		if err = cmd.Process.Kill(); err != nil {
			log.Println("failed to kill: %s, error: %s", cmd.Path, err)
		}
		go func() {
			<-done // allow goroutine to exit
		}()
		log.Println("process:%s killed", cmd.Path)
		return err, true
	case err = <-done:
		return err, false
	}
}
