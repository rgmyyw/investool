// 获取历史股价

package eniu

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/axiaoxin-com/goutils"
	"github.com/axiaoxin-com/logging"
	"go.uber.org/zap"
)

// RespHistoricalStockPrice 历史股价接口返回结构
type RespHistoricalStockPrice struct {
	Date  []string  `json:"date"`
	Price []float64 `json:"price"`
}

// LastYearFinalPrice 获取去年12月份最后一个交易日的股价
func (p RespHistoricalStockPrice) LastYearFinalPrice() float64 {
	if len(p.Date) == 0 {
		return 0
	}
	for i := len(p.Date) - 1; i > 0; i-- {
		prefix := fmt.Sprintf("%d-12-", time.Now().Year()-1)
		date := p.Date[i]
		if strings.Contains(date, prefix) {
			price := p.Price[i]
			logging.Debugf(nil, "date:%s price:%f", date, price)
			return price
		}
	}
	return 0
}

// HistoricalVolatility 计算历史波动率
// 历史波动率计算方法：https://goodcalculators.com/historical-volatility-calculator/
// 1、从市场上获得标的股票在固定时间间隔(如每天DAY、每周WEEK或每月MONTH等)上的价格。
// 2、对于每个时间段，求出该时间段末的股价与该时段初的股价之比的自然对数。
// 3、求出这些对数值的标准差即为历史波动率的估计值
// 4、若将日、周等标准差转化为年标准差，需要再乘以一年中包含的时段数量的平方根(如，选取时间间隔为每天，则若扣除闭市，每年中有250个交易日，应乘以根号250)
func (p RespHistoricalStockPrice) HistoricalVolatility(ctx context.Context, period string) (float64, error) {
	priceLen := len(p.Price)
	if priceLen == 0 {
		return -1.0, errors.New("no historical price data")
	}
	// 求末初股价比自然对数
	logs := []float64{}
	for i := priceLen - 1; i >= 1; i-- {
		endPrice := p.Price[i]
		startPrice := p.Price[i-1]
		log := math.Log(endPrice / startPrice)
		logs = append(logs, log)
	}
	// 标准差
	stdev, err := goutils.StdDeviationFloat64(logs)
	if err != nil {
		return -1.0, err
	}
	logging.Debugs(ctx, "stdev:", stdev)

	periodValue := float64(250)
	period = strings.ToUpper(period)
	switch period {
	case "DAY":
		periodValue = 1
	case "WEEK":
		periodValue = 5
	case "MONTH":
		periodValue = 21.75
	case "YEAR":
		periodValue = 250
	}
	volatility := stdev * math.Sqrt(periodValue)
	// 数据异常时全部股价为 0 导致返回 NaN
	if math.IsNaN(volatility) {
		return -1, errors.New("volatility is NaN")
	}
	return volatility, nil
}

// QueryHistoricalStockPrice 获取历史股价（前复权），最新数据在最后，有一天的延迟
// 原数据源亿牛网 eniu.com 接口已停止服务（恒定 500），切换为腾讯行情历史 K 线接口
// （东财 push2his 节点对机房出口 IP 存在间歇性掐连接，不可依赖）
func (e Eniu) QueryHistoricalStockPrice(ctx context.Context, secuCode string) (RespHistoricalStockPrice, error) {
	pathCode := e.GetPathCode(ctx, secuCode)
	if pathCode == "" {
		return RespHistoricalStockPrice{}, errors.New("invalid secuCode " + secuCode)
	}
	// 800 个交易日（约 3.3 年）窗口：历史波动率与去年年末价均足够
	apiurl := fmt.Sprintf(
		"https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,day,,,800,qfq",
		pathCode,
	)
	logging.Debug(ctx, "QueryHistoricalStockPrice "+apiurl+" begin")
	beginTime := time.Now()
	resp := struct {
		Code int `json:"code"`
		Data map[string]struct {
			// 除权除息日的行末尾会多一个分红信息对象（第 7 元素），
			// 因此元素类型用 interface{} 而非 string
			QfqDay [][]interface{} `json:"qfqday"`
			Day    [][]interface{} `json:"day"`
		} `json:"data"`
	}{}
	err := goutils.HTTPGET(ctx, e.HTTPClient, apiurl, nil, &resp)
	latency := time.Now().Sub(beginTime).Milliseconds()
	logging.Debug(ctx, "QueryHistoricalStockPrice "+apiurl+" end", zap.Int64("latency(ms)", latency))

	if err != nil {
		return RespHistoricalStockPrice{}, err
	}
	if resp.Code != 0 {
		return RespHistoricalStockPrice{}, errors.New("kline api code != 0 for " + secuCode)
	}
	stockData, ok := resp.Data[pathCode]
	if !ok {
		return RespHistoricalStockPrice{}, errors.New("no kline data for " + secuCode)
	}
	// 个别标的返回键为 day（无复权数据），优先取 qfqday
	klines := stockData.QfqDay
	if len(klines) == 0 {
		klines = stockData.Day
	}
	// klines 元素形如 ["2026-10-13","1287.000","1274.890","1288.480","1268.110","26573.000"]
	// 字段依次为 日期/开盘/收盘/最高/最低/成交量，取收盘价
	// 前复权下早期年份可能因累积分红调整为非正数，跳过以免波动率对数收益计算出现 NaN
	result := RespHistoricalStockPrice{}
	for _, k := range klines {
		if len(k) < 3 {
			continue
		}
		date, ok1 := k[0].(string)
		priceStr, ok2 := k[2].(string)
		if !ok1 || !ok2 {
			continue
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			continue
		}
		result.Date = append(result.Date, date)
		result.Price = append(result.Price, price)
	}
	if len(result.Date) == 0 {
		return result, errors.New("no kline data for " + secuCode)
	}
	return result, nil
}

// GetPathCode 返回接口 url path 中的股票代码
func (e Eniu) GetPathCode(ctx context.Context, secuCode string) string {
	s := strings.Split(secuCode, ".")
	if len(s) != 2 {
		return ""
	}
	return strings.ToLower(s[1]) + s[0]
}
