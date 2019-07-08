package syncer

import (
	"time"
)

/*
 * 正常情况下该函数不返回
 * 它应该watch topoDir指定的目录，该目录为每一个集群存放了一个topology.json的json文件
 * 这些json文件指示了该集群包含哪些机器，以及该集群的deploy目录
 * 该函数需要针对每台机器，通过rsync将{user}@ip:{deploy}/log同步到targetDir指定的目录中，同步之后
 * 需要能够区分哪些日志属于哪些集群以及哪些组件，同步的时间间隔为interval指定的间隔，同步的带宽限制由bwlimit指定
 * rsync命令行参数支持bwlimit
 */
func Sync(topoDir string, targetDir string, interval time.Duration, bwlimit int) error {
	return nil
}
