// Package cron 定时任务
package cron

import (
	"time"

	"github.com/axiaoxin-com/investool/models"
	"github.com/axiaoxin-com/logging"
	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"
)

var (
	promSyncLabels = []string{
		"jobname",
	}
	promSyncError = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cron",
			Name:      "sync_error",
			Help:      "cron sync job error",
		}, promSyncLabels,
	)
)

// RunCronJobs 启动定时任务
func RunCronJobs(async bool) {
	timezone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logging.Errorf(nil, "RunCronJobs time LoadLocation error:%v, using Local timezone as default", err.Error())
		timezone, _ = time.LoadLocation("Local")
	}
	logging.Debugf(nil, "cron timezone:%v", timezone)
	sched := gocron.NewScheduler(timezone)

	// 全量同步任务（SyncFund/SyncFundManagers/SyncIndustryList）不在本进程中调度：
	// SearchFunds 全量 2.5 万基金峰值内存 >1G，与 web 服务同进程会触发 cgroup OOM
	// （上游 README 中的 oom 注释即此因），改由 sync-data 命令在独立容器中执行
	// 定时读取本地的JSON数据更新到全局变量
	sched.Cron(viper.GetString("app.cronexp.sync_global_vars")).Do(models.InitGlobalVars)

	if async {
		sched.StartAsync()
	} else {
		sched.StartBlocking()
	}
}
