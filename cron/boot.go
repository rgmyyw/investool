// 启动引导：首次运行时生成数据文件

package cron

import (
	"os"
	"path/filepath"

	"github.com/axiaoxin-com/investool/models"
	"github.com/axiaoxin-com/logging"
)

func fileExists(f string) bool {
	_, err := os.Stat(f)
	return err == nil
}

// EnsureDataFiles 检查数据文件是否齐全，缺失时立即强制同步一次
// 上游因部署机内存不足改为"json 由外部同步"，本部署内存充足，
// 由程序自己在启动时和定时任务中维护这些文件；
// 按文件缺失情况精确调度对应同步任务，避免重复全量拉取
func EnsureDataFiles() {
	if err := os.MkdirAll(filepath.Dir(models.IndustryListFilename), 0755); err != nil {
		logging.Error(nil, "EnsureDataFiles mkdir error:"+err.Error())
		return
	}

	if !fileExists(models.IndustryListFilename) {
		logging.Info(nil, "industry_list.json missing, syncing...")
		syncIndustryList(true)
	}
	if !fileExists(models.FundManagersFilename) {
		logging.Info(nil, "fund_managers.json missing, syncing...")
		syncFundManagers(true)
	}
	if !fileExists(models.FundAllListFilename) ||
		!fileExists(models.FundTypeListFilename) ||
		!fileExists(models.Fund4433ListFilename) {
		logging.Info(nil, "fund data files missing, syncing...")
		syncFund(true)
	}
	logging.Info(nil, "EnsureDataFiles done")
}
