package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	WEIBO_SUMMARY = "https://s.weibo.com/top/summary"
	MARTCH_RULE   = `<a href="(\/weibo\?q=[^"]+)".*?>(.+)<\/a>`
	STORE_DIR     = "store"

	README_HEAD = "# Weibo Hot List \n" +
		" ![Refresh](https://github.com/baiyutang/weibo-hot-list/workflows/Refresh/badge.svg)\n\n" +
		"微博话题爬虫小玩意，利用 Github Action 的调度脚本每一小时更新一次 \n\n " +
		"创意来自 [justjavac](https://github.com/justjavac/weibo-trending-hot-search)\n"
)

type newItem map[string]string

type newsList []newItem

func getFileNews() newsList {
	var todayNews newsList
	todayFile := getTodayFileName()
	_, err := os.Stat(todayFile)
	if err != nil {
		_, err := os.Stat(STORE_DIR)
		if err != nil {
			os.Mkdir(STORE_DIR, 0777)
		}
		f, err := os.Create(todayFile)
		if err != nil {
			fmt.Println("创建文件失败", todayFile, err)
		}
		defer f.Close()
	}
	news, err := ioutil.ReadFile(todayFile)
	if err != nil {
		return todayNews
	}
	json.Unmarshal(news, &todayNews)
	return todayNews
}

func getTodayFileName() string {
	now := time.Now()
	return STORE_DIR + "/" + now.Format("2006-01-02") + ".json"
}

func mergeList(old newsList, latest newsList) newsList {
	if len(latest) < 1 {
		return old
	}
	for _, item := range latest {
		find := false
		for _, v := range old {
			if v["title"] == item["title"] {
				find = true
			}
		}
		if !find {
			old = append(old, item)
		}
	}
	return old
}

func updateReadme() bool {
	all := getFileNews()
	if len(all) < 1 {
		return false
	}
	now := time.Now()
	content := README_HEAD + "## 微博今日热榜 更新于 " + now.Format("2006-01-02 15:04:05") + "\n"
	for _, news := range all {
		content = content + "1. [" + news["title"] + "](https://s.weibo.com/" + news["url"] + ")\n\n"
	}
	result := ioutil.WriteFile("README.md", []byte(content), 0666)
	if result != nil {
		fmt.Println("生成README文件失败", result)
	}
	return true
}

func main() {
	resp, err := http.Get(WEIBO_SUMMARY)
	if err != nil {
		fmt.Printf("请求发生错误：%v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("响应错误：%d", resp.StatusCode)
		os.Exit(2)
	}
	scanner := bufio.NewScanner(resp.Body)
	started := false
	var newsList newsList
	reg := regexp.MustCompile(MARTCH_RULE)
	for scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			break
		}
		line := scanner.Text()
		if strings.Contains(line, "<tbody>") {
			started = true
		}
		if started == true && strings.Contains(line, "</tbody>") {
			break
		}
		params := reg.FindStringSubmatch(line)
		if len(params) > 2 {
			item := map[string]string{
				"url":   params[1],
				"title": params[2],
			}
			newsList = append(newsList, item)
		}
	}
	all := mergeList(getFileNews(), newsList)
	str, _ := json.Marshal(all)
	result := ioutil.WriteFile(getTodayFileName(), str, 0666)
	if result != nil {
		fmt.Println("写入榜单数据失败", result)
	}
	updateReadme()
}
