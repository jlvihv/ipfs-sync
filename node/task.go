package node

import (
	"fmt"
	"master/docker"
	"master/ipfs"
	"os"
	"path/filepath"
	"time"
)

type Task struct {
	MainServer string         `json:"main_server,omitempty" form:"main_server"`
	Compose    docker.Compose `json:"compose" form:"compose"`
	IPFS       ipfs.IPFS      `json:"ipfs" form:"ipfs"`
}

func (t *Task) Execute(dir string) {
	fmt.Println("执行任务")
	if err := t.writeComposeToFile(); err != nil {
		return
	}
	if err := t.Compose.Start(dir); err != nil {
		fmt.Println(err)
		return
	}
}

func (t *Task) Download(dir string) error {
	if t.IPFS.Cid == "" {
		fmt.Println("任务中 cid 字段为空，视为无历史数据，不下载")
		return nil
	}
	fmt.Println("连接 ipfs")
	t.IPFS.Connect("http://localhost:5001")
	fmt.Println("从 ipfs 下载文件，cid:", t.IPFS.Cid)
	if err := t.IPFS.Get(t.IPFS.Cid, dir); err != nil {
		fmt.Println(err)
		return err
	}
	time.Sleep(10 * time.Second)
	return nil
}

func (t *Task) Upload(dir string, cidChan chan string) {
	time.Sleep(20 * time.Second) // 等 20 秒以后再上传
	t.IPFS.Connect("http://localhost:5001")
	for {
		path := filepath.Join(dir)
		cid, err := t.IPFS.AddDir(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		cidChan <- cid
		fmt.Printf("%s 中的文件已上传到 ipfs，cid: %s\n", path, cid)
		time.Sleep(10 * time.Second)
	}
}

func (t *Task) writeComposeToFile() error {
	// 把字符串形式的 compose 写入到文件中
	if err := os.WriteFile(t.Compose.Filename, []byte(t.Compose.Content), 0644); err != nil {
		fmt.Printf("写入 compose 文件失败: %s\n", err)
		return err
	}
	return nil
}
