package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// 定义响应结构体
type ResponseAlist struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RawURL string `json:"raw_url"`
	} `json:"data"`
}

// 定义请求数据结构体
type RequestData struct {
	Path     string `json:"path"`
	Password string `json:"password"`
}

// AlistUrl 函数返回 raw_url 和 error
func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path>", os.Args[0])
	}
	Images := os.Args[1] // 获取命令行参数

	// 网站链接
	url := "https://alist.master-jsx.top/api/fs/get"

	// 创建请求数据
	requestData := RequestData{
		Path:     Images,
		Password: "", // 这里可以根据需要设置密码
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	// 创建一个新的 POST 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	clientAlist := &http.Client{}
	resp, err := clientAlist.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	// 打印响应结果
	//fmt.Println("Response Status:", resp.Status)
	//fmt.Println("Response Body:", string(body))

	// 解析 JSON 响应
	var responsealist ResponseAlist
	err = json.Unmarshal(body, &responsealist)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	// 提取 raw_url 并打印
	fmt.Println(responsealist.Data.RawURL)
}
