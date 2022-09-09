package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"master/docker"
	"master/ipfs"
	"net/http"
	"os"
	"sync"
	"time"
)

type AsMaster struct {
	sync.Mutex
	MainServer  string // 主服务器，主服务器的文件会上传到ipfs,其他服务器只会从ipfs下载，而不会上传
	LastIpfsCid string
	Port        int
	registry    sync.Map // 注册表
	portOffset  int      // 端口偏移量，为了解决在同一台机器上启动多个节点，端口占用的问题
}

func (n *AsMaster) StartGin() {
	r := gin.Default()
	r.POST("/register", n.register)
	r.POST("/heartbeat", n.heartbeat)
	r.Run(fmt.Sprintf(":%d", n.Port))
}

func (n *AsMaster) CheckSlaveStatus() {
	for {
		n.checkSlaveStatus()
		time.Sleep(10 * time.Second)
	}
}

func (n *AsMaster) checkSlaveStatus() {
	n.registry.Range(func(key, value interface{}) bool {
		node := value.(*AsSlave)
		// 已离线的就不管了
		if node.Status != StatusOnline {
			return true
		}
		if time.Now().Unix()-node.LastHeartbeatTime > OverduePeriod*(node.OverdueTimes+1) {
			node.OverdueTimes += 1
			fmt.Printf("node %s overdue %d times\n", key, node.OverdueTimes)
		}
		if node.OverdueTimes >= MaximumOverdueTimes {
			node.Status = StatusOffline
			fmt.Printf("node %s is offline\n", key)
			// 如果离线的是主服务器，将 MainServer 字段置空，为防止多线程冲突，加锁
			n.Lock()
			if fmt.Sprintf("%s:%d", node.Host, node.Port) == n.MainServer {
				fmt.Println("主服务器已离线")
				n.MainServer = ""
				n.portOffset += 1
			}
			n.Unlock()
		}
		return true
	})
}

// DispatchTask 派发任务
func (n *AsMaster) DispatchTask() {
	fmt.Println("dispatch task")
	// 读取 compose 文件
	b, err := os.ReadFile("compose.yaml")
	if err != nil {
		fmt.Println("read compose file failed", err)
		return
	}
	composeContent := string(b)
	for {
		n.dispatchTask(composeContent, 40000+n.portOffset)
		time.Sleep(10 * time.Second)
	}
}

func (n *AsMaster) dispatchTask(composeContent string, port int) {
	// 构建 task
	task := &Task{
		MainServer: n.MainServer,
		Compose: docker.Compose{
			Content:  composeContent,
			Filename: "compose.yaml",
			Port:     port,
		},
		IPFS: ipfs.IPFS{
			Cid: n.LastIpfsCid,
		},
	}
	// 向从节点发送task请求
	n.registry.Range(func(key, value interface{}) bool {
		node := value.(*AsSlave)
		// 如果已离线或已有任务，就不派发任务
		if node.Status != StatusOnline || node.IsBusy {
			return true
		}
		// 发送请求
		body, _ := json.Marshal(task)
		fmt.Printf("send task to %s\n", key)
		resp, err := http.Post(fmt.Sprintf("http://%s/task", key), "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("send task to node failed: %s\n", err)
			return true
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("send task to node failed: %s\n", err)
			return true
		}
		fmt.Println(string(body))
		n.Lock()
		if n.MainServer == "" {
			n.MainServer = fmt.Sprintf("%s", key)
			fmt.Printf("%s 将作为主服务器\n", key)
		}
		n.Unlock()
		return true
	})
}

// 处理注册请求
func (n *AsMaster) register(c *gin.Context) {
	var slave = &AsSlave{}
	err := c.ShouldBindJSON(slave)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid node",
		})
		return
	}
	// 检查是否已注册
	slaveNode, ok := n.registry.Load(fmt.Sprintf("%s:%d", slave.Host, slave.Port))
	if ok {
		c.JSON(http.StatusOK, gin.H{
			"message": "node already registered",
		})
		n.refreshSlaveStatus(slaveNode.(*AsSlave), slave)
		return
	}
	// 注册
	n.registry.Store(fmt.Sprintf("%s:%d", slave.Host, slave.Port), slave)
	fmt.Printf("node %s:%d registered\n", slave.Host, slave.Port)
	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

// 处理心跳请求
func (n *AsMaster) heartbeat(c *gin.Context) {
	var slave = &AsSlave{}
	err := c.ShouldBindJSON(slave)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid node",
		})
		return
	}
	slaveNode, ok := n.registry.Load(fmt.Sprintf("%s:%d", slave.Host, slave.Port))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "node not found, you may need to register first",
		})
		return
	}
	n.refreshSlaveStatus(slaveNode.(*AsSlave), slave)
	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func (n *AsMaster) refreshSlaveStatus(slaveNode *AsSlave, slave *AsSlave) {
	slaveNode.LastHeartbeatTime = time.Now().Unix()
	slaveNode.OverdueTimes = 0
	slaveNode.Status = StatusOnline
	slaveNode.IsBusy = slave.IsBusy
	slaveNode.LastIpfsCid = slave.LastIpfsCid
	if fmt.Sprintf("%s:%d", slave.Host, slave.Port) == n.MainServer {
		n.LastIpfsCid = slave.LastIpfsCid
		if n.LastIpfsCid != "" {
			fmt.Printf("主服务器的 IPFS CID 更新为 %s\n", n.LastIpfsCid)
		}
	}
}
