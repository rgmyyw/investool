// 全量数据同步命令：在独立的一次性容器中执行，与 web 服务隔离内存

package cmds

import (
	"fmt"

	"github.com/axiaoxin-com/investool/cron"
	"github.com/urfave/cli/v2"
)

const (
	// ProcessorSyncData 全量数据同步
	ProcessorSyncData = "sync-data"
)

// CommandSyncData 全量同步基金/行业/基金经理数据到本地 JSON 文件
// SearchFunds 峰值内存超过 1G，须在高内存限额的独立容器中运行：
// docker compose run --rm sync
func CommandSyncData() *cli.Command {
	return &cli.Command{
		Name:  ProcessorSyncData,
		Usage: "全量同步数据文件（industry/fund/fund_managers），供 webserver 加载",
		Action: func(c *cli.Context) error {
			cron.SyncAllData()
			fmt.Println("sync-data done")
			return nil
		},
	}
}
