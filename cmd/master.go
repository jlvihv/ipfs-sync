package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"master/node"
)

var masterCmd = &cobra.Command{
	Use: "master",
	Run: func(cmd *cobra.Command, args []string) {
		masterRun()
	},
}

func masterRun() {
	fmt.Println("作为主节点启动")
	n := node.Node{Identity: node.MasterNode, AsMaster: node.AsMaster{Port: masterPort}}
	// 启动主节点的gin
	go n.AsMaster.StartGin()
	// 周期性检查从节点状态
	go n.AsMaster.CheckSlaveStatus()
	// 向空闲节点派发任务
	go n.AsMaster.DispatchTask()
	// 阻塞，使其不退出
	select {}
}
