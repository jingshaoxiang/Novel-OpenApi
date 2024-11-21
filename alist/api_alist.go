package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RawURL string `json:"raw_url"`
	} `json:"data"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <image_name>")
		return
	}

	imageName := os.Args[1]
	url := "https://alist.master-jsx.top/api/fs/get"
	payload := map[string]string{
		"path":     "/" + imageName,
		"password": "",
	}

	// 将请求体编码为JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// 创建新的HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:132.0) Gecko/20100101 Firefox/132.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Referer", "https://alist.master-jsx.top/?page=1")
	req.Header.Set("Authorization", "")
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Origin", "https://alist.master-jsx.top")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", "_ga_L7WEXVQCR9=GS1.1.1730338898.1.1.1730339413.0.0.0; _ga=GA1.1.1306726360.1730338898; _ga_89WN60ZK2E=GS1.1.1732156357.8.1.1732156855.0.0.0; p_uv_id=5cbce50c066af2b253988bd29c7b06ac")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Priority", "u=0")
	req.Header.Set("TE", " trailers")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// 解析JSON响应
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// 输出raw_url
	if response.Code == 200 {
		fmt.Print(response.Data.RawURL)
	} else {
		fmt.Print("Error:", response.Message)
	}
}
