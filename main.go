package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

var used_ports []int
var online_sandbox_ids []string

func main() {

	router := gin.Default()

	_, err := client.NewClientWithOpts()
	if err != nil {
		fmt.Println("Docker Client Error: ", err)
	}

	router.GET("/", home)
	router.GET("/create", create)
	router.POST("/remove", remove)

	// 환경변수에 SAN_PORT가 있으면 이용 없으면 5000

	env := os.Getenv("SAN_PORT")
	if env == "" {
		env = "5000"
	}

	router.Run(":" + env)
}
