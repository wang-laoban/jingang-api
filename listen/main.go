package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestPayload 定义请求参数的结构体
type RequestPayload struct {
	Area          string  `json:"area"`
	Location      string  `json:"location"`
	Bend          *bool   `json:"bend"`
	CameraName    string  `json:"cameraName"`
	CameraNo      string  `json:"cameraNo"`
	Type          string  `json:"type"`
	WarningTime   string  `json:"warningTime"`
	Longitude     float64 `json:"longitude"`
	Latitude      float64 `json:"latitude"`
	Effectiveness string  `json:"effectiveness"`
	AttaID        int     `json:"attaId"`
	AlarmPosition string  `json:"alarmPosition"`
	Render        string  `json:"render"`
}

func main() {
	r := gin.Default()

	// 定义 POST 接口
	r.POST("/jjg/anon/videoAI/warning/save", func(c *gin.Context) {
		var payload RequestPayload

		// 绑定 JSON 数据
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		fmt.Println("payload:", payload)

		// 模拟处理逻辑（可以在此添加实际业务逻辑）
		c.JSON(http.StatusOK, gin.H{
			"message": "Request received successfully",
			"data":    payload,
		})
	})

	// 启动服务
	r.Run(":8080")
}
