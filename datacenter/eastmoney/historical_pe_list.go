// 获取历史市盈率

package eastmoney

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/axiaoxin-com/goutils"
	"github.com/axiaoxin-com/logging"
	"go.uber.org/zap"
)

// RespHistoricalPE 历史市盈率接口返回结构
type RespHistoricalPE struct {
	Data [][]struct {
		Securitycode string `json:"SECURITYCODE"`
		Datetype     string `json:"DATETYPE"`
		Sl           string `json:"SL"`
		Endate       string `json:"ENDATE"`
		Value        string `json:"VALUE"`
	} `json:"data"`
	Pe [][]struct {
		Securitycode string `json:"SECURITYCODE"`
		Pe30         string `json:"PE30"`
		Pe50         string `json:"PE50"`
		Pe70         string `json:"PE70"`
		Total        string `json:"TOTAL"`
		Rn1          string `json:"RN1"`
		Rn2          string `json:"RN2"`
		Rn3          string `json:"RN3"`
	} `json:"pe"`
}

// HistoricalPE 历史 pe
type HistoricalPE struct {
	Value float64
	Date  string
}

// HistoricalPEList 历史 pe 列表
type HistoricalPEList []HistoricalPE

// GetMidValue 获取历史 pe 中位数
func (h HistoricalPEList) GetMidValue(ctx context.Context) (float64, error) {
	values := []float64{}
	for _, i := range h {
		values = append(values, i.Value)
	}
	return goutils.MidValueFloat64(values)
}

// QueryHistoricalPEList 获取历史市盈率
// 原 emfront APP 接口已下线（返回反爬字体内容），切换为
// datacenter-web 估值分析数据（RPT_VALUEANALYSIS_DET，PE_TTM 按交易日）
func (e EastMoney) QueryHistoricalPEList(ctx context.Context, secuCode string) (HistoricalPEList, error) {
	code := ""
	if s := strings.Split(secuCode, "."); len(s) == 2 {
		code = s[0]
	} else {
		code = secuCode
	}
	apiurl := fmt.Sprintf(
		"https://datacenter-web.eastmoney.com/api/data/v1/get?reportName=RPT_VALUEANALYSIS_DET&columns=TRADE_DATE%%2CPE_TTM&filter=(SECURITY_CODE%%3D%%22%s%%22)&sortColumns=TRADE_DATE&sortTypes=1&pageSize=5000&pageNumber=1",
		code,
	)
	logging.Debug(ctx, "EastMoney QueryHistoricalPEList "+apiurl+" begin")
	beginTime := time.Now()
	resp := struct {
		Result struct {
			Data []struct {
				TradeDate string  `json:"TRADE_DATE"`
				PETTM     float64 `json:"PE_TTM"`
			} `json:"data"`
		} `json:"result"`
	}{}
	err := goutils.HTTPGET(ctx, e.HTTPClient, apiurl, nil, &resp)
	latency := time.Now().Sub(beginTime).Milliseconds()
	logging.Debug(
		ctx,
		"EastMoney QueryHistoricalPEList "+apiurl+" end",
		zap.Int64("latency(ms)", latency),
	)
	if err != nil {
		return nil, err
	}
	result := HistoricalPEList{}
	if len(resp.Result.Data) == 0 {
		return nil, errors.New("no historical pe data")
	}
	for _, i := range resp.Result.Data {
		date := i.TradeDate
		// TRADE_DATE 形如 "2018-01-02 00:00:00"，截取日期部分
		if len(date) > 10 {
			date = date[:10]
		}
		pe := HistoricalPE{
			Date:  date,
			Value: i.PETTM,
		}
		result = append(result, pe)
	}
	return result, nil
}
