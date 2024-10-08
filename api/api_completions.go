package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Config 定义配置文件结构体
type Config struct {
	Parameters struct {
		ParamsVersion               int         `yaml:"params_version"`
		Width                       int         `yaml:"width"`
		Height                      int         `yaml:"height"`
		Scale                       int         `yaml:"scale"`
		Sampler                     string      `yaml:"sampler"`
		Steps                       int         `yaml:"steps"`
		NSamples                    int         `yaml:"n_samples"`
		UCPreset                    int         `yaml:"ucPreset"`
		QualityToggle               bool        `yaml:"qualityToggle"`
		SM                          bool        `yaml:"sm"`
		SMDyn                       bool        `yaml:"sm_dyn"`
		DynamicThresholding         bool        `yaml:"dynamic_thresholding"`
		ControlNetStrength          float64     `yaml:"controlnet_strength"`
		Legacy                      bool        `yaml:"legacy"`
		AddOriginalImage            bool        `yaml:"add_original_image"`
		CFGRescale                  int         `yaml:"cfg_rescale"`
		NoiseSchedule               string      `yaml:"noise_schedule"`
		LegacyV3Extend              bool        `yaml:"legacy_v3_extend"`
		SkipCFGAboveSigma           interface{} `yaml:"skip_cfg_above_sigma"`
		DeliberateEulerAncestralBug bool        `yaml:"deliberate_euler_ancestral_bug"`
		PreferBrownian              bool        `yaml:"prefer_brownian"`
	} `yaml:"parameters"`
}

// Choice 定义响应结构体
type Choice struct {
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"`
	Index        int     `json:"index"`
	Logprobs     *string `json:"logprobs"`
}

type Delta struct {
	Content string  `json:"content"`
	Refusal *string `json:"refusal"`
	Role    string  `json:"role"`
}

type Response struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             *string  `json:"usage"`
}

// ChatRequest 定义请求结构体
type ChatRequest struct {
	Authorization string    `json:"Authorization"`
	Messages      []Message `json:"messages"`
	Model         string    `json:"model"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 默认出图
var defaultPositiveWords = "blue eyes，white hair，big breasts，{expressionless:2.0}，indifference，{double bun:2.0}，navel，cleavage，{breast curtain:3.0}，{JiangShi girly:4.0}，Eyes detail，detached sleeves，hair over one eye，{squatting:3.0}，{Open leg:3.0},best quality, amazing quality, very aesthetic, absurdres"
var defaultNegativeWords = "nsfw, lowres, {bad}, error, fewer, extra, missing, worst quality, jpeg artifacts, bad quality, watermark, unfinished, displeasing, chromatic aberration, signature, extra digits, artistic error, username, scan, [abstract], low quality,worst quality,normal quality,text,signature,jpeg artifacts,bad anatomy,old,early,copyright name,watermark,artist name,signature,, 2girls,cat tail,cat,cherry blossoms tree,"

// 提取正词和反词的函数
func extractWords(userInput string) (string, string) {
	re := regexp.MustCompile(`正词(.+?)\s*反词(.+)`)
	matches := re.FindStringSubmatch(userInput)

	if len(matches) != 3 {
		log.Println("未找到正词和反词，使用默认配置")
		return defaultPositiveWords, defaultNegativeWords
	}

	positiveWords := strings.Split(matches[1], "，")
	negativeWords := strings.Split(matches[2], ", ")

	// 将切片连接成字符串
	positiveWordsStr := strings.Join(positiveWords, ", ")
	negativeWordsStr := strings.Join(negativeWords, ", ")

	return positiveWordsStr, negativeWordsStr
}

// 提取链接的函数
func extractLinks(userInput string) []string {
	re := regexp.MustCompile(`https?://[^\s]+`)
	matches := re.FindAllString(userInput, -1)
	return matches
}

// Completions 处理请求的函数
func Completions(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

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

	// 解析 YAML 配置文件
	byteValue, _ := ioutil.ReadAll(configFile)
	var config Config
	if err := yaml.Unmarshal(byteValue, &config); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	// 图片存放地址
	DiskDir := viper.GetString("disk.dir")

	// 图片映射地址
	AlistDir := viper.GetString("alist.dir")

	// 教程地址
	drawingTutorial := viper.GetString("drawingTutorial.url")

	// 如果是 OPTIONS 请求，直接返回 200 OK
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析请求体
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取最后一条用户输入
	var userInput string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userInput = req.Messages[i].Content
			log.Printf("User input found: %s", userInput)
			break
		}
	}

	// 提取用户输入中的链接
	imageURL := extractLinks(userInput)
	var base64String string
	if len(imageURL) > 0 {
		// 选择第一个提取到的链接
		imageURLS := imageURL[0]
		// 解析图片为bash
		base64String, err = ImageURLToBase64(imageURLS)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	positiveWords, negativeWords := extractWords(userInput)
	fmt.Println("正词:", positiveWords)
	fmt.Println("反词:", negativeWords)

	// 生成一个随机种子
	rand.Seed(time.Now().UnixNano()) // 使用当前时间的纳秒数作为随机数生成器的种子
	randomSeed := rand.Intn(1000000) // 生成一个0到999999之间的随机数

	// 创建请求到目标 API
	apiURL := "https://image.novelai.net/ai/generate-image"
	log.Println("Preparing payload for API request.")
	// 支持自定义
	payload := map[string]interface{}{
		"input":  positiveWords + ",best quality, amazing quality, very aesthetic, absurdres",
		"model":  req.Model,
		"action": "generate",
		"parameters": map[string]interface{}{
			"params_version":                 config.Parameters.ParamsVersion,
			"width":                          config.Parameters.Width,
			"height":                         config.Parameters.Height,
			"scale":                          config.Parameters.Scale,
			"sampler":                        config.Parameters.Sampler,
			"steps":                          config.Parameters.Steps,
			"seed":                           randomSeed,
			"n_samples":                      config.Parameters.NSamples,
			"ucPreset":                       config.Parameters.UCPreset,
			"qualityToggle":                  config.Parameters.QualityToggle,
			"sm":                             config.Parameters.SM,
			"sm_dyn":                         config.Parameters.SMDyn,
			"dynamic_thresholding":           config.Parameters.DynamicThresholding,
			"controlnet_strength":            config.Parameters.ControlNetStrength,
			"legacy":                         config.Parameters.Legacy,
			"add_original_image":             config.Parameters.AddOriginalImage,
			"cfg_rescale":                    config.Parameters.CFGRescale,
			"noise_schedule":                 config.Parameters.NoiseSchedule,
			"legacy_v3_extend":               config.Parameters.LegacyV3Extend,
			"skip_cfg_above_sigma":           config.Parameters.SkipCFGAboveSigma,
			"negative_prompt":                negativeWords + "pussy, nipples, nude, naked, nsfw, lowres, {bad}, error, fewer, extra, missing, worst quality, jpeg artifacts, bad quality, watermark, unfinished, displeasing, chromatic aberration, signature, extra digits, artistic error, username, scan, [abstract]",
			"deliberate_euler_ancestral_bug": config.Parameters.DeliberateEulerAncestralBug,
			"prefer_brownian":                config.Parameters.PreferBrownian,
		},
	}

	// 根据是否有有效的 base64String 来决定是否添加这三个字段
	if base64String != "" {
		payload["parameters"].(map[string]interface{})["reference_image_multiple"] = []interface{}{base64String}
		payload["parameters"].(map[string]interface{})["reference_information_extracted_multiple"] = []interface{}{1}
		payload["parameters"].(map[string]interface{})["reference_strength_multiple"] = []interface{}{0.6}
	}

	// 将 payload 转换为 JSON
	payloadBytes, _ := json.Marshal(payload)
	log.Println("Payload marshaled to JSON")

	// 创建新的请求
	client := &http.Client{}
	request, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Failed to create new request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("API request created successfully:", request)

	// 设置请求头
	request.Header.Set("Authorization", r.Header.Get("Authorization"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Origin", "https://novelai.net")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("Referer", "https://novelai.net/")
	log.Println("Request headers set.")
	//fmt.Println("Authorization", r.Header.Get("Authorization"))

	// 发送请求
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Printf("Received response with status code: %d", resp.StatusCode)

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to generate image: "+resp.Status, resp.StatusCode)
		log.Printf("Error from API: %s", resp.Status)
		return
	}

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to read response body: %v", err)
		return
	}
	log.Println("Response body read successfully.")

	// 创建 ZIP 读取器
	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		http.Error(w, "Failed to read ZIP file: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to create zip reader: %v", err)
		return
	}
	log.Println("ZIP file read successfully.")

	// 确保保存图像的目录存在
	outputDir := DiskDir
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		http.Error(w, "Failed to create directory: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to create output directory: %v", err)
		return
	}
	log.Printf("Output directory created or already exists: %s", outputDir)

	// 获取当前时间戳
	timestamp := time.Now().Unix()
	imageName := fmt.Sprintf("%d_temp.png", timestamp)
	imagePath := outputDir + "/" + imageName
	log.Printf("Image will be saved as: %s", imagePath)

	// 提取指定的图像文件并进行流式输出
	for _, file := range zipReader.File {
		if file.Name == "image_0.png" { // 根据实际文件名进行匹配
			dstFile, err := os.Create(fmt.Sprintf("%s/%s", outputDir, imageName))
			if err != nil {
				http.Error(w, "创建图像文件失败: "+err.Error(), http.StatusInternalServerError)
				log.Printf("创建图像文件失败: %v", err)
				return
			}
			defer dstFile.Close()

			// 打开 ZIP 中的文件
			srcFile, err := file.Open()
			if err != nil {
				http.Error(w, "打开 ZIP 中的文件失败: "+err.Error(), http.StatusInternalServerError)
				log.Printf("打开 ZIP 中的文件失败: %v", err)
				return
			}

			// 将图像写入目标文件
			if _, err := io.Copy(dstFile, srcFile); err != nil {
				http.Error(w, "写入图像文件失败: "+err.Error(), http.StatusInternalServerError)
				log.Printf("写入图像文件失败: %v", err)
				return
			}
			log.Println("图像文件写入成功。")

			//获取解析链接
			//alistURL := alist.AlistUrl(imagePath)

			// 进行流式输出
			publicLink := fmt.Sprintf("您需要的图片在这里👉🏻 [点击预览](%s/%s) * [玩家画廊](%s) * [画图教程](%s)", AlistDir, imageName, AlistDir, drawingTutorial)
			fmt.Println(publicLink)

			// 组装流式输出数据
			sseResponse := fmt.Sprintf(
				"data: {\"id\":\"%s\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"%s\"},\"logprobs\":null,\"finish_reason\":null}]}\n\n",
				"chatcmpl-"+fmt.Sprintf("%d", timestamp), // 生成一个唯一的 id
				timestamp,
				req.Model,
				publicLink,
			)

			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte(sseResponse))
			w.(http.Flusher).Flush() // 刷新响应缓冲区到客户端
			break
		}
	}

	// 结束流式输出
	w.Write([]byte("event: end\n\n"))
	w.(http.Flusher).Flush() // 刷新最后一条消息
}

// 启用 CORS
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}
