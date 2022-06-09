package main

import (
	"fmt"

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

	router.Run(":5000")
}
