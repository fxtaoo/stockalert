package conf

import (
	"fmt"
	"stockalert/stock"
	"strconv"
	"strings"
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
	Chan             chan struct{} `json:"-"`
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
	c.Chan = make(chan struct{}, 1)
	return file.JsonInitValue("conf.json", c)
}

func (c *Conf) Save() error {
	return file.JsonSaveValue("conf.json", c)
}

// 更新按钮
func (c *Conf) StockUpdate(ticker, option string) string {
	// 阻塞
	c.Chan <- struct{}{}
	defer func() { <-c.Chan }()

	updateResult := ""
	tickerIndex := 0

	// ticker 下标
	for ; tickerIndex < len(c.Stocks); tickerIndex++ {
		if c.Stocks[tickerIndex].Ticker == ticker {
			break
		}
	}
	if tickerIndex == len(c.Stocks) {
		tickerIndex = -1
	}

	switch option {
	case "alertmail":
		// 股票修改提醒
		c.Stocks[tickerIndex].AlertMail = !c.Stocks[tickerIndex].AlertMail
		if c.Stocks[tickerIndex].AlertMail {
			updateResult = fmt.Sprintf("%s %s %s",
				c.Stocks[tickerIndex].Ticker,
				c.Stocks[tickerIndex].Name,
				"开启邮件提醒",
			)
		} else {
			updateResult = fmt.Sprintf("%s %s %s",
				c.Stocks[tickerIndex].Ticker,
				c.Stocks[tickerIndex].Name,
				"关闭邮件提醒",
			)
		}
	case "new":
		// 股票新增
		if tickerIndex != -1 {
			// 重复
			updateResult = fmt.Sprintf("新增失败 %s %s 以存在！", ticker, c.Stocks[tickerIndex].Name)
		} else {
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
		}
	case "del":
		// 股票删除
		tmpStocks := []stock.Stock{}
		for i := range c.Stocks {
			if i != tickerIndex {
				tmpStocks = append(tmpStocks, c.Stocks[i])
			}
		}
		updateResult = fmt.Sprintf("%s %s 以删除", ticker, c.Stocks[tickerIndex].Name)
		c.Stocks = tmpStocks
	}

	// 保存
	if err := c.Save(); err != nil {
		updateResult = fmt.Sprintf("配置保存失败：%s", err.Error())
	}

	return updateResult
}

// 股票移动
func (c *Conf) StockMove(ticker string) string {
	indexs := strings.Split(ticker, ">")
	index1, err := strconv.Atoi(indexs[0])
	if err != nil {
		return fmt.Sprintf("%s 输出有误：%s", ticker, err.Error())
	}
	index2, err := strconv.Atoi(indexs[1])
	if err != nil {
		return fmt.Sprintf("%s 输出有误：%s", ticker, err.Error())
	}

	if index1 < 1 || index1 > len(c.Stocks) {
		return fmt.Sprintf("%s 输出有误：%d 超出范围", ticker, index1)
	}

	if index2 < 1 || index2 > len(c.Stocks) {
		return fmt.Sprintf("%s 输出有误：%d 超出范围", ticker, index2)
	}

	// 0 开始的数组下标
	index1 -= 1
	index2 -= 1

	tmpStocks := []stock.Stock{}

	// 删除掉 index1
	for i := range c.Stocks {
		if i != index1 {
			tmpStocks = append(tmpStocks, c.Stocks[i])
		}
	}

	tmpStocks2 := []stock.Stock{}
	for i := range tmpStocks {
		if i == index2 {
			tmpStocks2 = append(tmpStocks2, c.Stocks[index1])
		}
		tmpStocks2 = append(tmpStocks2, tmpStocks[i])
	}

	// 为最后一个
	if len(tmpStocks) == len(tmpStocks2) {
		tmpStocks2 = append(tmpStocks2, c.Stocks[index1])
	}

	c.Stocks = tmpStocks2
	return fmt.Sprintf("序号 %d %s 移动至 %d", index1+1, c.Stocks[index1].Name, index2+1)
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
