// 股票价值提醒
package main

import (
	"fmt"
	"html/template"
	"log"
	"stockalert/conf"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	// 初始化配置
	conf := conf.Conf{}
	err := conf.Init()
	if err != nil {
		log.Fatal(err)
	}

	// 计算全部股票 ROE
	calcStocks := func(chus string) {
		// 增加网页获取间隔
		ticker := time.NewTicker(time.Second)
		for i := range conf.Stocks {
			if strings.Contains(chus, conf.Stocks[i].CHUS) {
				err := conf.Stocks[i].CalcValue(
					conf.Alert.Low,
					conf.Alert.High,
				)
				if err != nil {
					fmt.Println(err)
				}
				<-ticker.C
			}
		}
		ticker.Stop()
	}
	// 初始化 A 股美股
	conf.StocksCalcCHTime = time.Now()
	conf.StocksCalcUSTime = time.Now()
	calcStocks("chus")

	// 定时运行
	cron := cron.New()
	// A 股
	cron.AddFunc(conf.Alert.CronCH, func() {
		conf.StocksCalcCHTime = time.Now()
		calcStocks("ch")
		conf.StocksAlertMail("ch")
	})
	// 美股
	cron.AddFunc(conf.Alert.CronUS1, func() {
		conf.StocksCalcUSTime = time.Now()
		calcStocks("us")
		conf.StocksAlertMail("us")
	})
	// 美股
	cron.AddFunc(conf.Alert.CronUS2, func() {
		conf.StocksCalcUSTime = time.Now()
		calcStocks("us")
		conf.StocksAlertMail("us")
	})
	cron.Start()

	// 更新结果
	updateResult := make(chan string)

	r := gin.Default()
	r.Use(gin.BasicAuth(gin.Accounts(conf.Web.Auth)))

	r.SetFuncMap(template.FuncMap{
		"floatFormat": func(f float64) string {
			return fmt.Sprintf("%.2f", f)
		},
		"alertMailFormat": func(a bool) string {
			if a {
				return "开启"
			} else {
				return "关闭"
			}
		},
		"timeFormat": func(t time.Time) string {
			return fmt.Sprint(t.Format("2006/01/02 15:04:05"))
		},
		"floatIndex": func(i int) int {
			return i + 1
		},
	})
	r.LoadHTMLGlob("templates/*.html")
	r.GET("/tool/stock", func(ctx *gin.Context) {
		ctx.HTML(200, "stock.html", conf)
	})
	r.GET("/tool/stock/updateresult", func(ctx *gin.Context) {
		ctx.String(200, "%s", <-updateResult)
		updateResult <- ""
	})
	r.POST("/tool/stock", func(ctx *gin.Context) {
		ticker := ctx.PostForm("ticker")
		if ticker == "" {
			ticker = ctx.Query("ticker")
		}
		option := ctx.Query("option")
		quit := make(chan int)
		go func() {
			// post updateresult 失败避免锁死
			for {
				select {
				case <-quit:
					return
				case <-time.After(time.Second * 3):
					<-updateResult
					updateResult <- ""
				}
			}

		}()
		// 位置插入，格式 2>1
		if strings.Contains(ticker, ">") {
			updateResult <- conf.StockMove(ticker)
		} else {
			updateResult <- conf.StockUpdate(ticker, option)
		}

		<-updateResult
		// 结束协程
		quit <- 0
		// 延迟避免不出现弹窗
		<-time.After(time.Millisecond * 1500)
		ctx.HTML(200, "stock.html", conf)
	})

	r.Run(":" + conf.Web.Port)
}
