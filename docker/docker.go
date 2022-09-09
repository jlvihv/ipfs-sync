package docker

import (
	"fmt"
	"master/tool"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type Compose struct {
	Content  string `json:"content,omitempty" form:"content"`   // compose 文件内容
	Filename string `json:"filename,omitempty" form:"filename"` // 文件名
	Port     int    `json:"port,omitempty" form:"port"`         // 使用的端口
	//StartCommand string `json:"start_command,omitempty" form:"start_command"` // 启动命令
}

func (c *Compose) Start(dir string) error {
	fmt.Println("启动 docker compose")
	tool.CheckAndCreateDir(dir)
	command := "docker compose up -d"
	if c.Filename != "" {
		command = fmt.Sprintf("docker compose -f %s up -d", c.Filename)
	}
	cmdArr := strings.Split(command, " ")
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", c.Port))
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("CURRENT_UID=%s:%s", currentUser.Uid, currentUser.Gid))
	// 重定向输出到系统标准输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
