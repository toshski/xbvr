package scrape

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/xbapps/xbvr/pkg/models"
)

type ScrapeHttpKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type ScrapeHttpCookieDetail struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
	Host   string `json:"host"`
}
type ScrapeHttpConfig struct {
	Headers []ScrapeHttpKeyValue     `json:"headers"`
	Cookies []ScrapeHttpCookieDetail `json:"cookies"`
	Body    string                   `json:"body"`
}

func SetupHtmlRequest(kvKey string, req *http.Request) *http.Request {
	conf := GetScrapeHttpConfig(kvKey)
	for _, header := range conf.Headers {
		req.Header.Add(header.Key, header.Value)
	}
	for _, cookie := range conf.Cookies {
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value, Domain: cookie.Domain, Path: cookie.Path})
	}
	if conf.Body != "" {
		body := strings.NewReader(conf.Body)
		req.Body = io.NopCloser(body)
	}
	return req
}
func SetupCollector(kvKey string, collector *colly.Collector) *colly.Collector {
	conf := GetScrapeHttpConfig(kvKey)
	// setup header for the OnRequest Function
	if len(conf.Headers) > 0 {
		collector.OnRequest(func(r *colly.Request) {
			for _, header := range conf.Headers {
				r.Headers.Set(header.Key, header.Value)
			}
		})
	}
	// setup cookies
	for _, cookie := range conf.Cookies {
		c := collector.Cookies(cookie.Host)
		newcookie := http.Cookie{Name: cookie.Name, Value: cookie.Value, Domain: cookie.Domain, Path: cookie.Path}
		c = append(c, &newcookie)
		collector.SetCookies(cookie.Host, c)
	}
	return collector

}
func GetScrapeHttpConfig(kvKey string) ScrapeHttpConfig {
	db, _ := models.GetCommonDB()

	c := ScrapeHttpConfig{}
	var kv models.KV
	kv.Key = kvKey
	db.Find(&kv)
	json.Unmarshal([]byte(kv.Value), &c)
	return c
}

func SaveScrapeHttpConfig(kvKey string, config ScrapeHttpConfig) {
	var kv models.KV
	kv.Key = kvKey
	jsonStr, _ := json.MarshalIndent(config, "", "  ")
	kv.Value = string(jsonStr)
	kv.Save()
}
