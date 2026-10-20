// 启动引导：首次运行时生成数据文件

package cron

import (
	"os"
	"path/filepath"

	"github.com/axiaoxin-com/investool/models"
	"github.com/axiaoxin-com/logging"
)

// EnsureDataFiles 检查数据文件是否齐全，缺失时立即强制同步一次
// 上游因部署机内存不足改为"json 由外部同步"，本部署内存充足，
// 由程序自己在启动时和定时任务中维护这些文件
func EnsureDataFiles() {
	files := []string{
		models.IndustryListFilename,
		models.FundAllListFilename,
		models.Fund4433ListFilename,
		models.FundTypeListFilename,
		models.FundManagersFilename,
	}
	missing := false
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			missing = true
			break
		}
	}
	if !missing {
		return
	}

	if err := os.MkdirAll(filepath.Dir(models.IndustryListFilename), 0755); err != nil {
		logging.Error(nil, "EnsureDataFiles mkdir error:"+err.Error())
		return
	}
	logging.Info(nil, "data files missing, running initial sync...")
	syncIndustryList(true)
	syncFundManagers(true)
	syncFund(true)
}
