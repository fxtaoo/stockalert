package conf

import (
	"fmt"
	"stockalert/stock"
	"time"

	"github.com/fxtaoo/golib/file"
	"github.com/fxtaoo/golib/mail"
)

type Conf struct {
	Smtp  mail.Smtp `json:"smtp"`
	Alert Alert     `json:"alert"`
	Web   struct {
		Auth map[string]string `json:"auth"`
		Port string            `json:"port"`
	} `json:"web"`
	Stocks           []stock.Stock `json:"stocks"`
	StocksCalcCHTime time.Time     `json:"-"`
	StocksCalcUSTime time.Time     `json:"-"`
}

type Alert struct {
	Mails             []string `json:"mails"`
	CronCH            string   `json:"cronCH"`
	CronUS1           string   `json:"cronUS1"`
	CronUS2           string   `json:"cronUS2"`
	Low               float64  `json:"low"`
	High              float64  `json:"high"`
	AlarmIntervalTime float64  `json:"alarmIntervalTime"`
}

func (c *Conf) Init() error {
	return file.JsonInitValue("conf.json", c)
}

func (c *Conf) Save() error {
	return file.JsonSaveValue("conf.json", c)
}

// 更新按钮
func (c *Conf) StockUpdate(ticker, alertmail string) string {
	updateResult := ""

	if alertmail == "on" {
		// 股票修改提醒
		for i := range c.Stocks {
			if c.Stocks[i].Ticker == ticker {
				c.Stocks[i].AlertMail = !c.Stocks[i].AlertMail

				if c.Stocks[i].AlertMail {
					updateResult = fmt.Sprintf("%s %s %s",
						c.Stocks[i].Ticker,
						c.Stocks[i].Name,
						"开启邮件提醒",
					)
				} else {
					updateResult = fmt.Sprintf("%s %s %s",
						c.Stocks[i].Ticker,
						c.Stocks[i].Name,
						"关闭邮件提醒",
					)
				}
			}
			if updateResult == "" {
				updateResult = fmt.Sprintf("%s 不存在", ticker)
			}
		}

	} else {
		// 股票新增删除
		tmpStocks := []stock.Stock{}
		delStockName := ""

		for i := range c.Stocks {
			if c.Stocks[i].Ticker == ticker {
				// 股票删除
				delStockName = c.Stocks[i].Name
				continue
			}
			tmpStocks = append(tmpStocks, c.Stocks[i])
		}

		// 股票新增
		if len(c.Stocks) == len(tmpStocks) {
			newStock := stock.Stock{
				Ticker:    ticker,
				AlertMail: true,
			}
			err := newStock.CalcValue(c.Alert.Low, c.Alert.High)
			if err != nil {
				updateResult = fmt.Sprintf("%s 添加失败：%s", ticker, err.Error())
				return updateResult
			}
			c.Stocks = append(c.Stocks, newStock)
			updateResult = fmt.Sprintf("%s %s 添加成功", ticker, newStock.Name)
		} else {
			// 股票删除
			c.Stocks = c.Stocks[:0]
			c.Stocks = append(c.Stocks, tmpStocks...)
			updateResult = fmt.Sprintf("%s %s 以删除", ticker, delStockName)
		}
	}

	// 保存
	if err := c.Save(); err != nil {
		updateResult = fmt.Sprintf("配置保存失败：%s", err.Error())
	}

	return updateResult
}

func (c *Conf) StocksAlertMail() {
	content := ""
	timeNow := time.Now()

	// 添加内容
	contentAdd := func(index int) {
		lh := ""
		if c.Stocks[index].Value < c.Alert.Low {
			lh = "低估"
		} else {
			lh = "高估"
		}
		content = fmt.Sprintf("%s%s %.2f %s<br>",
			content,
			c.Stocks[index].Name,
			c.Stocks[index].Value,
			lh,
		)
		c.Stocks[index].AlertMailTime = timeNow
	}

	for index, stock := range c.Stocks {
		if stock.AlertMail && (stock.Value > c.Alert.High || stock.Value < c.Alert.Low) {
			// 没有发送过提醒邮件
			if stock.AlertMailTime.IsZero() {
				contentAdd(index)
			} else {
				// 发送过邮件，大于间隔时间
				if time.Since(stock.AlertMailTime).Minutes() > c.Alert.AlarmIntervalTime {
					contentAdd(index)
				}
			}
		}
	}

	if content != "" {
		// 邮件
		mail := mail.Mail{
			To:      c.Alert.Mails,
			Subject: "股票价值提醒",
			Body:    content,
		}

		mail.SendAlone(&c.Smtp)
	}
}
