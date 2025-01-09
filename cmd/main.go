package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/xuri/excelize/v2"
)

// MQTT 消息结构体
// 根据实际的消息格式，调整了结构体的字段
// 从 MQTT 消息中获取的参数：modeName, deviceCode, deviceId, deviceName, modelCode, image_url 等
var topic = "video/gateway/edge/video/alarm/#"
var isDebug bool
var cameraMap = make(map[string]CameraData)

type CameraData struct {
	Area       string `json:"area"`
	Location   string `json:"location"`
	CameraName string `json:"cameraName"`
}

var ModelMap = map[string]string{
	"sidewalk":           "video_Al_warn_016",
	"human_damoyolo":     "video_Al_warn_017",
	"peron_with_truck":   "video_Al_warn_018",
	"peron_with_around":  "video_Al_warn_019",
	"cigarette_damoyolo": "video_Al_warn_020",
	"helmet_damoyolo":    "video_Al_warn_021",
	"safety_rope":        "video_Al_warn_022",
	"half_mask":          "video_Al_warn_023",
	"crossing_road":      "video_Al_warn_024",
	"sleeve":             "video_Al_warn_025",
	"phone_damoyolo":     "video_Al_warn_026",
	"cars":               "video_Al_warn_027",
	"ecar":               "video_Al_warn_028",
}

// video_Al_warn_016 未按人行道行走 sidewalk
// video_Al_warn_017 人员逗留  human_damoyolo
// video_Al_warn_018 人员登高车辆	peron_with_truck
// video_Al_warn_019 人员对车侧及车位操作识别  peron_with_around
// video_Al_warn_020 吸烟	cigarette_damoyolo
// video_Al_warn_021 未戴安全帽	helmet_damoyolo
// video_Al_warn_022 登高作业识别安全绳防坠器	safety_rope
// video_Al_warn_023 半面罩未戴识别	half_mask
// video_Al_warn_024 横穿道路  crossing_road
// video_Al_warn_025 工装检测(长短袖)	sleeve
// video_Al_warn_026 接打电话	phone_damoyolo
// video_Al_warn_027 车辆长时间停留	cars
// video_Al_warn_028 电动车禁入识别	ecar

//	{
//	    "params": "{\"modeName\":\"\u4eba\u5458\u767b\u9ad8\u8f66\u8f86\",\"deviceCode\":\"TCC_C_HKSXT_01\",\"deviceId\":201562414658998,\"deviceName\":\"\u505c\u8f66\u573a+240\u53f7\u6444\u50cf\u673a+\u67aa\u673a+\u505c\u8f66\u573aC\u533a\u5165\u53e3\",\"modelCode\":\"peron_with_truck\"}",
//	    "pub_name": "video/gateway/edge/model/alarm/201562414658998",
//	    "type": 1,
//	    "token": "1735887631261",
//	    "key": "201562414658998",
//	    "code": 200,
//	    "msg": "predict success",
//	    "detail": "peron_with_truck/202501/recognized_1735888795_8718774.jpg",
//	    "mode": "peron_with_truck",
//	    "image_url": "http://192.168.108.5:8080/video-gateway/original/2025/01/03/1735887631261_201562414658998.jpg",
//	    "videoPath": "video/201562414658998/peron_with_truck/2025-01-03/alarm_1735887637.mp4"
//	}
type Params struct {
	ModeName   string `json:"modeName"`
	DeviceCode string `json:"deviceCode"`
	DeviceID   int    `json:"deviceId"`
	DeviceName string `json:"deviceName"`
	ModelCode  string `json:"modelCode"`
}

type MQTTMessage struct {
	ParamInfo Params `json:"param_info"`
	ParamsStr string `json:"params"`
	PubName   string `json:"pub_name"`
	Type      int    `json:"type"`
	Token     string `json:"token"`
	Key       string `json:"key"`
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Detail    string `json:"detail"`
	Mode      string `json:"mode"`
	ImageURL  string `json:"image_url"`
	VideoPath string `json:"videoPath"`
}

// API 请求结构体，接收处理后需要提交的请求数据
type RequestPayload struct {
	Area        string `json:"area"`        // 区域
	Location    string `json:"location"`    // 位置
	CameraName  string `json:"cameraName"`  // 摄像头名称
	CameraNo    string `json:"cameraNo"`    // 摄像头编号
	Type        string `json:"type"`        //模型类型
	WarningTime string `json:"warningTime"` //告警时间
	Render      string `json:"render"`      // 图片 base64
}

var apiURL = "http://172.41.166.150:7002/jjg/anon/videoAI/warning/save"

// var imgFix = "http://192.168.108.5:8080/video-gateway/"
var imgFix = "http://192.168.108.2:8080/video-gateway/"

func main() {
	//获取程序运行的flag 参数判断是否需要打印
	isDebug := os.Getenv("DEBUG")
	// flag.BoolVar(&isDebug, "debug", false, "Enable debug mode")
	// flag.Parse()
	fmt.Println("Debug mode:", isDebug)

	filePath := "./data.xlsx"
	//获取url中的图片，并将图片转为base64
	err := LoadExcelData(filePath)
	if err != nil {
		log.Fatalf("Error loading Excel data: %v", err)
	}
	fmt.Println(cameraMap)
	startMQTTClient()
	<-make(chan struct{})
}

func startMQTTClient() {
	broker := "tcp://192.168.108.5:1883"
	// topic := "video/gateway/edge/video/alarimgStrm/#"
	clientID := "golang-mqtt-client"

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetUsername("mediaai")
	opts.SetPassword("video_media!Q@W#E")
	opts.SetClientID(clientID)
	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("Connected to MQTT broker")
		if token := c.Subscribe(topic, 1, messageHandler); token.Wait() && token.Error() != nil {
			log.Fatalf("Failed to subscribe to topic: %v", token.Error())
		}
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("Connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
}

// 处理收到的 MQTT 消息
func messageHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message on topic %s: %s", msg.Topic(), string(msg.Payload()))

	// 解析消息内容
	var mqttMsg MQTTMessage
	if err := json.Unmarshal(msg.Payload(), &mqttMsg); err != nil {
		log.Printf("Failed to parse MQTT message: %v", err)
		return
	}
	mqttMsg.DecodeParams()

	// 获取图片Base64
	// img := GetImageFromUrl("https://k.sinaimg.cn/n/sinacn10109/653/w640h813/20190414/0e44-hvscktf5572580.png/w700d1q75cms.jpg")
	img := GetImageFromUrl(imgFix + mqttMsg.Detail)
	imgStr, err := ImageToBase64(img)
	if err != nil {
		log.Fatal(err)
	}
	code := mqttMsg.ParamInfo.DeviceCode
	data, exists := cameraMap[code]
	if !exists {
		fmt.Printf("Code %s not found\n", code)
		return
	}

	// 构建 API 请求数据
	payload := RequestPayload{
		Area:        data.Area,
		Location:    data.Location,
		CameraName:  data.CameraName,
		CameraNo:    code,
		Type:        ModelMap[mqttMsg.Mode],
		WarningTime: time.Now().Format("2006-01-02 15:04:05"),
		Render:      imgStr, // 图片是 base64 格式
	}

	// 发送数据到接口
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal request payload: %v", err)
		return

	}

	if isDebug {
		fmt.Println("POST:", string(jsonData))
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create HTTP request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	clientHTTP := &http.Client{}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("Data sent successfully")
	} else {
		log.Printf("Failed to send data, status code: %d", resp.StatusCode)
	}
}

func (m *MQTTMessage) DecodeParams() error {
	if err := json.Unmarshal([]byte(m.ParamsStr), &m.ParamInfo); err != nil {
		return err
	}
	return nil
}

func GetImageBase64(m MQTTMessage) string {
	return m.ImageURL
}

func GetImageFromUrl(url string) []byte {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read the data into a byte slice
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func ImageToBase64(imageData []byte) (string, error) {
	// Encode the bytes to a Base64 string
	base64String := base64.StdEncoding.EncodeToString(imageData)
	return base64String, nil
}

func LoadExcelData(filePath string) error {
	// Open the Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Get all rows in the first sheet
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}

	// Iterate over the rows (skip the header row)
	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}
		if len(row) < 4 {
			log.Printf("Skipping incomplete row at index %d: %v\n", i, row)
			continue
		}

		// Populate the map with code as the key
		code := row[0]
		cameraMap[code] = CameraData{
			Area:       row[1],
			Location:   row[2],
			CameraName: row[3],
		}
	}

	return nil
}
