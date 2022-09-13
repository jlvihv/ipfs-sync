package cmd

import (
	"fmt"
	"master/node"

	"github.com/spf13/cobra"
)

var slaveCmd = &cobra.Command{
	Use: "slave",
	Run: func(cmd *cobra.Command, args []string) {
		slaveRun()
	},
}

func slaveRun() {
	fmt.Println("作为从节点启动")
	n := node.Node{Identity: node.SlaveNode, AsSlave: node.AsSlave{
		MasterAddr: masterAddr,
		Host:       host,
		Port:       port,
	}}
	// 向主节点发送注册请求
	n.AsSlave.SendRegisterRequest()
	// 周期性向主节点发送心跳
	go n.AsSlave.SendHeartbeatRequestPeriodically()
	// 启动从节点的 gin，用来接收任务
	go n.AsSlave.StartGin()
	// 阻塞，使其不退出
	select {}
}
