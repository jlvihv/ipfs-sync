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
		n.registry.Range(func(key, value interface{}) bool {
			node := value.(*AsSlave)
			// 已离线的就不管了
			if node.Status != StatusOnline {
				return true
			}
			if time.Now().Unix()-node.LastHeartbeatTime > OverduePeriod*(node.OverdueTimes+1) {
				node.OverdueTimes += 1
				fmt.Printf("node %s:%d overdue %d times\n", node.Host, node.Port, node.OverdueTimes)
			}
			if node.OverdueTimes >= MaximumOverdueTimes {
				node.Status = StatusOffline
				fmt.Printf("node %s:%d is offline\n", node.Host, node.Port)
				// 如果离线的是主服务器，将 MainServer 字段置空
				if fmt.Sprintf("%s:%d", node.Host, node.Port) == n.MainServer {
					fmt.Println("主服务器已离线")
					n.Lock()
					n.MainServer = ""
					n.portOffset += 1
					n.Unlock()
				}
				n.registry.Delete(fmt.Sprintf("%s:%d", node.Host, node.Port))
			}
			return true
		})
		time.Sleep(10 * time.Second)
	}
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
		// 构建 task
		task := &Task{
			Compose: docker.Compose{
				Content:  composeContent,
				Filename: "compose-1.yaml",
				Port:     40000 + n.portOffset,
				//StartCommand: "docker compose up -d",
			},
			IPFS: ipfs.IPFS{},
		}
		// 向从节点发送task请求
		if n.MainServer == "" {
			task.NeedUpload = true
			task.NeedExecute = true
			if n.LastIpfsCid != "" {
				task.NeedDownload = true
				task.IPFS.Cid = n.LastIpfsCid
				task.IPFS.Filename = fmt.Sprintf("%d", task.Compose.Port)
			}
		} else {
			task.NeedUpload = false
			task.NeedExecute = false
			task.NeedDownload = false
			task.IPFS.Cid = n.LastIpfsCid
		}
		n.registry.Range(func(key, value interface{}) bool {
			node := value.(*AsSlave)
			if node.Status != StatusOnline || node.IsBusy {
				//fmt.Printf("node %s:%d is busy or offline\n", node.Host, node.Port)
				return true
			}
			// 发送请求
			body, _ := json.Marshal(task)
			fmt.Printf("send task to %s:%d\n", node.Host, node.Port)
			resp, err := http.Post(fmt.Sprintf("http://%s:%d/task", node.Host, node.Port), "application/json", bytes.NewBuffer(body))
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
				fmt.Printf("%s:%d 将作为主服务器\n", node.Host, node.Port)
				n.MainServer = fmt.Sprintf("%s:%d", node.Host, node.Port)
			}
			n.Unlock()
			return true
		})
		time.Sleep(10 * time.Second)
	}
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

type RelationshipGraph struct {
	Master Node
	Nodes  []Node
}
