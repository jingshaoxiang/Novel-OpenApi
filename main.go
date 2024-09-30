package main

import (
	"NoveAI3/api"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/v1/chat/completions", api.Completions) // 修改了路由
	log.Println("Starting server on :3388")
	if err := http.ListenAndServe(":3388", nil); err != nil {
		log.Fatal(err)
	}
}
