package node

import (
	"fmt"
	"master/docker"
	"master/ipfs"
	"master/tool"
	"os"
	"path/filepath"
	"time"
)

type Task struct {
	NeedUpload   bool
	NeedDownload bool
	NeedExecute  bool
	Compose      docker.Compose `json:"compose" form:"compose"`
	IPFS         ipfs.IPFS      `json:"ipfs" form:"ipfs"`
}

func (t *Task) Execute() {
	fmt.Println("执行任务")
	if err := t.writeComposeToFile(); err != nil {
		return
	}
	if err := t.Compose.Start(); err != nil {
		fmt.Println(err)
		return
	}
}

func (t *Task) Download() {
	t.IPFS.Connect("http://localhost:5001")
	for {
		if err := t.IPFS.Get(t.IPFS.Cid); err != nil {
			fmt.Println(err)
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (t *Task) Upload() {
	time.Sleep(20 * time.Second) // 等20秒以后再上传
	t.IPFS.Connect("http://localhost:5001")
	for {
		path := filepath.Join("./data", fmt.Sprintf("%d", t.Compose.Port))
		tool.CheckAndCreateDir(path)
		cid, err := t.IPFS.AddDir(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		t.IPFS.Cid = cid
		fmt.Printf("%s 中的文件已上传到 ipfs，cid: %s\n", path, cid)
		time.Sleep(10 * time.Second)
	}
}

func (t *Task) writeComposeToFile() error {
	// 把字符串形式的compose写入到文件中
	if err := os.WriteFile(t.Compose.Filename, []byte(t.Compose.Content), 0644); err != nil {
		fmt.Printf("写入 compose 文件失败: %s\n", err)
		return err
	}
	return nil
}
