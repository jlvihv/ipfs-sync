package ipfs

import (
	"fmt"
	ipfsapi "github.com/ipfs/go-ipfs-api"
	"master/tool"
	"path/filepath"
)

type IPFS struct {
	lastCid  string
	sh       *ipfsapi.Shell
	Cid      string `json:"cid,omitempty" form:"cid"`
	Filename string `json:"filename,omitempty" form:"filename"`
}

// Connect 连接并获取 ipfs shell
func (i *IPFS) Connect(url string) {
	i.sh = ipfsapi.NewShell(url)
}

// AddDir 把文件夹添加到 ipfs 中
func (i *IPFS) AddDir(dir string) (cid string, err error) {
	if i.sh == nil {
		return "", fmt.Errorf("ipfs shell 未连接")
	}
	cid, err = i.sh.AddDir(dir)
	//fmt.Printf("已添加文件夹到 ipfs，cid: %s\n", cid)
	return
}

func (i *IPFS) Get(cid string) error {
	if i.sh == nil {
		return fmt.Errorf("ipfs shell 未连接")
	}
	if cid == i.lastCid {
		return nil
	}
	outDir := filepath.Join("./data/ipfs", cid)
	if i.Filename != "" {
		outDir = filepath.Join("./data", i.Filename)
	}
	tool.CheckAndCreateDir(outDir)
	err := i.sh.Get(cid, outDir)
	if err != nil {
		return err
	}
	i.lastCid = cid
	fmt.Printf("已从 ipfs 获取文件夹，cid: %s\n", cid)
	return nil
}
