package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"time"
)

type AsSlave struct {
	MasterAddr        string     `json:"master_addr,omitempty"`         // 主节点地址
	Host              string     `json:"host,omitempty"`                // 从节点host
	Port              int        `json:"port,omitempty"`                // 从节点端口
	Status            StatusCode `json:"status,omitempty"`              // 从节点状态
	LastHeartbeatTime int64      `json:"last_heartbeat_time,omitempty"` // 从节点最后一次心跳时间
	OverdueTimes      int64      `json:"overdue_times,omitempty"`       // 从节点逾期次数
	LastIpfsCid       string     `json:"last_ipfs_cid,omitempty"`       // 从节点最后一次上传的 ipfs cid
	IsBusy            bool       `json:"is_busy,omitempty"`             // 从节点是否忙碌
}

// StartGin 启动从节点的 gin，用来接收任务
func (n *AsSlave) StartGin() {
	r := gin.Default()
	r.POST("/task", n.task)
	r.Run(fmt.Sprintf(":%d", n.Port))
}

// 处理task
func (n *AsSlave) task(c *gin.Context) {
	var t Task
	err := c.ShouldBindJSON(&t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("task is invalid: %s", err),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
	fmt.Printf("接收到主服务器cid: %s\n", t.IPFS.Cid)
	// 如果需要下载，就得等到下载完成
	if t.NeedDownload {
		n.IsBusy = true
		fmt.Println("开始下载")
		t.Download()
	}
	if t.NeedExecute {
		n.IsBusy = true
		go t.Execute()
	}
	if t.NeedUpload {
		go t.Upload()
	}
	go func(t *Task) {
		for {
			n.updateLastCid(t.IPFS.Cid)
			time.Sleep(10 * time.Second)
		}
	}(&t)
}

// SendRegisterRequest 发送注册请求
func (n *AsSlave) SendRegisterRequest() {
	fmt.Println("send register request")
	body, _ := json.Marshal(n)
	resp, err := http.Post(fmt.Sprintf("http://%s/register", n.MasterAddr), "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("注册失败: %s\n", err)
		return
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("注册失败: %s\n", err)
		return
	}
	fmt.Println(string(body))
}

// SendHeartbeatRequest 发送心跳请求
func (n *AsSlave) SendHeartbeatRequest() {
	fmt.Println("send heartbeat request")
	body, _ := json.Marshal(n)
	resp, err := http.Post(fmt.Sprintf("http://%s/heartbeat", n.MasterAddr), "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("发送心跳请求失败: %s\n", err)
		return
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("发送心跳请求失败: %s\n", err)
		return
	}
	fmt.Println(string(body))
}

// SendHeartbeatRequestPeriodically 周期性发送心跳请求
func (n *AsSlave) SendHeartbeatRequestPeriodically() {
	for {
		n.SendHeartbeatRequest()
		time.Sleep(OverduePeriod * time.Second)
	}
}

func (n *AsSlave) updateLastCid(cid string) {
	n.LastIpfsCid = cid
}
