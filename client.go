package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/colinyl/lib4go/security/md5"
	"github.com/colinyl/lib4go/utility"
)

var minSEQValue uint64 = 100000

type httpClient struct {
	client *http.Client
	data   *dataBlock
}

func NewHttpClient(data *dataBlock) *httpClient {
	return &httpClient{client: createClient(), data: data}
}

func (c *httpClient) Reqeust() (r *response) {
	defer func() {
		if err := recover(); nil != err && err.(error) != nil {
			log.Fatal(err.(error).Error())
			r = &response{success: false, url: c.data.URL, useTime: 0}
		}
	}()
	url := fmt.Sprintf("%s?%s", c.data.URL, c.makeParams())
	startTime := time.Now()
	resp, er := c.client.Get(url)
	endTime := time.Now()
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	success, er := c.ResultChanHanlde(body)
	if !success {
		log.Errorf("%s\n%s\n", url, string(body))

	}
	if er != nil {
		log.Error(er)
	}
	return &response{success: resp.StatusCode == 200 && er == nil && success,
		url: c.data.URL, useTime: subTime(startTime, endTime)}
}

func (h *httpClient) makeParams() string {
	if h.data.Params == nil || len(h.data.Params) == 0 {
		return ""
	}
	var (
		rawFormat string
		keys      []string
	)
	for k := range h.data.Params {
		if strings.HasPrefix(k, "$") {
			rawFormat = h.data.Params[k]
			continue
		} else {
			reg := regexp.MustCompile(`^[a-zA-Z]$`)
			var buffer []byte
			buffer = append(buffer, k[0])
			if reg.Match(buffer) {
				keys = append(keys, k)
			}
		}
	}
	sort.Sort(sort.StringSlice(keys))
	var keyValues []string
	var urlParams []string
	nmap := h.getDatamap()
	fullParamsMap := utility.NewDataMap()
	for _, k := range keys {
		var value string
		if v1, ok := h.data.Params[k]; ok {
			value = nmap.Translate(v1)
		} else {
			continue
		}
		fullParamsMap.Set(k, value)
		keyValues = append(keyValues, k+value)
		urlParams = append(urlParams, k+"="+value)
	}
	fullParamsMap.Set("raw", strings.Join(keyValues, ""))
	fullRaw := strings.Replace(fullParamsMap.Translate(rawFormat), " ", "", -1)
	urlParams = append(urlParams, "sign="+md5.Encrypt(fullRaw))
	return strings.Join(urlParams, "&")
}

func subTime(startTime time.Time, endTime time.Time) int {
	return int(endTime.Sub(startTime).Nanoseconds() / 1000 / 1000)
}
func (h *httpClient) getDatamap() utility.DataMap {
	mp, err := h.getFromChan()
	if err != nil {
		panic(err)
	}
	baseData := utility.NewDataMaps(mp)
	baseData.Set("guid", utility.GetGUID())
	baseData.Set("seq", fmt.Sprintf("%d", atomic.AddUint64(&minSEQValue, 1)))
	baseData.Set("timestamp", time.Now().Format("20060102150405"))
	baseData.Set("unixtime", fmt.Sprintf("%d", time.Now().Unix()))
	baseData.Set("uxmillisecond", fmt.Sprintf("%d", time.Now().Unix()*1000))
	return baseData
}

func createClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, 0)
				if err != nil {
					log.Fatal("timeout")
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   0,
			ResponseHeaderTimeout: 0,
		},
	}
}
