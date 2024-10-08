package main

import (
	"NoveAI3/api"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
)

func main() {
	//获取图片保存路径
	viper.SetConfigFile("config.yml")

	// 读取配置文件
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
		return
	} else {
		//fmt.Println("配置文件加载成功.......")
	}
	// 读取配置文件
	configFile, err := os.Open("config.yml")
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	defer configFile.Close()

	Port := viper.GetString("start.port")

	http.HandleFunc("/v1/chat/completions", api.Completions) // 修改了路由
	log.Println("Starting server on :" + Port)
	if err := http.ListenAndServe(":"+Port, nil); err != nil {
		log.Fatal(err)
	}
}
