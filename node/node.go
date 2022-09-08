package node

type StatusCode = int
type Identity = string

const (
	StatusOnline StatusCode = iota
	StatusOffline
)
const (
	MasterNode Identity = "master"
	SlaveNode  Identity = "slave"
)

const (
	OverduePeriod       = 10 // 节点逾期周期，单位秒
	MaximumOverdueTimes = 3  // 节点最大逾期次数，超过该次数则认为节点已离线
)

// Node 节点
type Node struct {
	Identity Identity // 节点身份
	AsMaster AsMaster // 作为主节点
	AsSlave  AsSlave  // 作为从节点
}
