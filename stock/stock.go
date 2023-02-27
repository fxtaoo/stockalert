package stock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type Stock struct {
	Ticker        string    `json:"ticker"`
	Name          string    `json:"name"`
	AlertMail     bool      `json:"alertmail"`
	AlertMailTime time.Time `json:"-"`
	Value         float64   `json:"-"`
	ValueTime     time.Time `json:"-"`
	ValueCSS      string    `json:"-"`
	PE            float64   `json:"-"`
	ROE           float64   `json:"-"`
	Price         string    `json:"-"`
	Dividend      string    `json:"-"`
	CHUS          string    `json:"chus"`
	URL           string    `json:"-"`
}

// PE（TTM）、名字 腾讯证券
func (s *Stock) GetPEName() (float64, error) {
	url := "https://gu.qq.com"

	switch s.Ticker[0:2] {
	case "60":
		url = fmt.Sprintf("%s/sh%s/gp", url, s.Ticker)
	case "00":
		url = fmt.Sprintf("%s/sz%s/gp", url, s.Ticker)
	}

	webDate := struct {
		Name  string `selector:"div.gb_title div.title_bg h1:nth-child(1)"`
		Price string `selector:"#spFP div:nth-child(1) span:nth-child(1)"`
		PE    string `selector:"div.content div.col-2 ul:nth-child(3) li:nth-child(4) span:nth-child(2)"`
	}{}

	err := GetWebData(
		"#hqpanel",
		url,
		&webDate,
	)

	if err != nil {
		return 0, fmt.Errorf("%s 获取\"PE、名字、价格\"数据异常", s.Ticker)
	}
	if webDate.Name == "" || webDate.PE == "" {
		return 0, fmt.Errorf("%s 获取\"PE、名字、价格\"数据异常", s.Ticker)
	}
	s.Name = webDate.Name
	s.Price = webDate.Price
	return strconv.ParseFloat(webDate.PE, 64)
}

// 五年平均 ROE 股息（集思录）
func (s *Stock) GetROEAVG() (float64, error) {
	s.URL = fmt.Sprintf("https://www.jisilu.cn/data/stock/%s", s.Ticker)

	webDate := struct {
		DividendAVG string `selector:"tr:nth-child(2) td:nth-child(2)" attr:"title"`
		ROEAVG      string `selector:"tr:nth-child(3) td:nth-child(2)"`
	}{}

	err := GetWebData(
		"#stock_detail tbody",
		s.URL,
		&webDate,
	)

	if err != nil {
		return 0, fmt.Errorf("%s 获取\"五年平均 ROE\"数据异常", s.Ticker)
	}
	if webDate.ROEAVG == "" || webDate.DividendAVG == "" {
		return 0, fmt.Errorf("%s 获取\"五年平均 ROE 股息\"数据异常", s.Ticker)
	}

	tmpStr := strings.Split(webDate.DividendAVG, "：")[1]
	if tmpStr == "-" {
		s.Dividend = "N/A"
	} else {
		s.Dividend = fmt.Sprintf("%s%%", tmpStr[:len(tmpStr)-2])
	}

	tmpStr = strings.Split(webDate.ROEAVG, " ")[1]
	if tmpStr == "-" {
		return 0, nil
	} else {
		return strconv.ParseFloat(tmpStr[:len(tmpStr)-1], 64)
	}
}

// 预测 ROE（亿牛网）
func (s *Stock) GetROEGuess() (float64, error) {
	url := "https://eniu.com/gu"

	switch s.Ticker[0:2] {
	case "60":
		url = fmt.Sprintf("%s/sh%s/roe", url, s.Ticker)
	case "00":
		url = fmt.Sprintf("%s/sz%s/roe", url, s.Ticker)
	}

	webDate := struct {
		ROEGuess string `selector:"p:nth-child(6) a"`
	}{}

	err := GetWebData(
		"#changyong",
		url,
		&webDate,
	)
	if err != nil {
		return 0, fmt.Errorf("%s 获取\"加权 ROE\"数据异常", s.Ticker)
	}
	if webDate.ROEGuess == "" {
		return 0, fmt.Errorf("%s 获取\"加权 ROE\"数据异常", s.Ticker)
	}
	return strconv.ParseFloat(webDate.ROEGuess[:len(webDate.ROEGuess)-1], 64)
}

// A 股计算价值
func (s *Stock) CalcValueCH() error {
	pe, err := s.GetPEName()
	if err != nil {
		return err
	}
	s.PE = pe
	roeavg, err := s.GetROEAVG()
	if err != nil {
		return err
	}
	ROEGuess, err := s.GetROEGuess()
	if err != nil {
		return err
	}

	if roeavg == 0 {
		s.ROE = ROEGuess
	} else {
		s.ROE = roeavg*0.7 + ROEGuess*0.3
	}

	s.Value = s.PE / s.ROE
	return nil
}

// 美股计算价值
func (s *Stock) CalcValueUS() error {
	// yahoo finance
	s.URL = fmt.Sprintf("https://finance.yahoo.com/quote/%s?p=%s", s.Ticker, s.Ticker)

	url := fmt.Sprintf("https://finance.yahoo.com/quote/%s/key-statistics?p=%s", s.Ticker, s.Ticker)

	webDate := struct {
		Name        string `selector:"#quote-header-info div:nth-child(2) div:nth-child(1) div:nth-child(1) h1"`
		Price       string `selector:"#quote-header-info div:nth-child(3) div:nth-child(1) div:nth-child(1) fin-streamer:nth-child(1)"`
		TrailingPE  string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(3) td:nth-child(2)"`
		ForwardPE   string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(4) td:nth-child(2)"`
		ROE         string `selector:"#Col1-0-KeyStatistics-Proxy section div:nth-child(3) div:nth-child(3) div div:nth-child(3) div div table tbody tr:nth-child(2) td:nth-child(2)"`
		TrailingPE1 string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(3) td:nth-child(3)"`
		TrailingPE2 string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(3) td:nth-child(4)"`
		TrailingPE3 string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(3) td:nth-child(5)"`
		TrailingPE4 string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(3) td:nth-child(6)"`
		TrailingPE5 string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(3) td:nth-child(7)"`
		ForwardPE1  string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(4) td:nth-child(3)"`
		ForwardPE2  string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(4) td:nth-child(4)"`
		ForwardPE3  string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(4) td:nth-child(5)"`
		ForwardPE4  string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(4) td:nth-child(6)"`
		ForwardPE5  string `selector:"#Col1-0-KeyStatistics-Proxy thead+tbody tr:nth-child(4) td:nth-child(7)"`
		Dividend    string `selector:"#Col1-0-KeyStatistics-Proxy section div:nth-child(3) div:nth-child(2) div div:nth-child(3) div div table tbody tr:nth-child(5) td:nth-child(2)"`
	}{}

	if err := GetWebData(
		"#app",
		url,
		&webDate,
	); err != nil || webDate.Name == "" || webDate.TrailingPE == "" || webDate.ForwardPE == "" || webDate.ROE == "" || webDate.Dividend == "" {
		return fmt.Errorf("%s 获取数据失败", s.Ticker)
	}

	// 计算股息
	calcPE := func(currentPE, pe1, pe2, pe3, pe4, pe5 string) (float64, error) {
		parseFloat := func(pe string, nonZeroNum *int) (float64, error) {
			if pe == "N/A" || pe == "" {
				return 0, nil
			} else {
				*nonZeroNum += 1
				return strconv.ParseFloat(pe, 64)
			}
		}
		if currentPE == "N/A" {
			// 有几个非零值
			nonZeroNum := new(int)
			pe1v, err := parseFloat(pe1, nonZeroNum)
			if err != nil {
				return 0, err
			}
			pe2v, err := parseFloat(pe2, nonZeroNum)
			if err != nil {
				return 0, err
			}
			pe3v, err := parseFloat(pe3, nonZeroNum)
			if err != nil {
				return 0, err
			}
			pe4v, err := parseFloat(pe4, nonZeroNum)
			if err != nil {
				return 0, err
			}
			pe5v, err := parseFloat(pe5, nonZeroNum)
			if err != nil {
				return 0, err
			}
			if *nonZeroNum == 0 {
				return 0, nil
			} else {
				return (pe1v + pe2v + pe3v + pe4v + pe5v) / float64(*nonZeroNum), nil
			}

		} else {
			return strconv.ParseFloat(currentPE, 64)
		}
	}

	trailingPE, err := calcPE(
		webDate.TrailingPE,
		webDate.TrailingPE1,
		webDate.TrailingPE2,
		webDate.TrailingPE3,
		webDate.TrailingPE4,
		webDate.TrailingPE5,
	)
	if err != nil {
		return err
	}

	forwardPE, err := calcPE(
		webDate.ForwardPE,
		webDate.ForwardPE1,
		webDate.ForwardPE2,
		webDate.ForwardPE3,
		webDate.ForwardPE4,
		webDate.ForwardPE5,
	)
	if err != nil {
		return err
	}
	s.Name = strings.TrimSpace(strings.Split(webDate.Name, "(")[0])
	s.PE = trailingPE*0.7 + forwardPE*0.3
	if webDate.ROE != "N/A" {
		roe, err := strconv.ParseFloat(webDate.ROE[:len(webDate.ROE)-1], 64)
		if err != nil {
			return err
		}
		s.ROE = roe
		s.Value = s.PE / s.ROE
	} else {
		s.ROE = -1
		s.Value = -1
	}

	s.Price = webDate.Price
	if s.Dividend != "N/A" {
		s.Dividend = fmt.Sprintf("%s%%", webDate.Dividend)
	}
	return nil
}

// 计算价值
func (s *Stock) CalcValue(low, high float64) error {
	if CHUS(s.Ticker[0]) {
		if err := s.CalcValueCH(); err != nil {
			return err
		}
		s.CHUS = "ch"
	} else {
		if err := s.CalcValueUS(); err != nil {
			return err
		}
		s.CHUS = "us"
	}

	s.ValueTime = time.Now()

	if s.Value < low {
		if s.Value == -1 {
			s.ValueCSS = "na"
		} else {
			s.ValueCSS = "low"
		}
	} else if s.Value > high {
		s.ValueCSS = "high"
	} else {
		s.ValueCSS = "middle"
	}

	return nil
}

// 从 web 获取数据
func GetWebData(selectors, url string, v interface{}) error {
	c := colly.NewCollector()
	c.OnHTML(selectors, func(e *colly.HTMLElement) {
		e.Unmarshal(v)
	})
	if err := c.Visit(url); err != nil {
		return err
	}
	return nil
}

// 从网络 API 获取数据
func GetWebAPI(url string, v interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return err
	}
	return nil
}

// ture A 股,false美股
func CHUS(c byte) bool {
	return strings.Contains("603", string(c))
}
