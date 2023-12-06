package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func adminRouter(admin *gin.RouterGroup) {
	admin.GET("/", func(c *gin.Context) {
		chall, _ := GetAllChall()
		RenderTemplates(c, gin.H{
			"challenges": chall,
		}, "admin")
	})
	admin.GET("/online", func(c *gin.Context) {
		online := GetOnlineSandbox()

		resp := strings.Join(online, "<br />")

		c.String(http.StatusOK, resp)
	})

	admin.POST("/image/add", createContainerHandler)

	admin.POST("/image/del", removeContainerHandler)
}

func createContainerHandler(c *gin.Context) {
	containerName := c.PostForm("containerName")
	dockerImage := c.PostForm("dockerImage")

	// Validate input (you can add more validation logic)
	if containerName == "" || dockerImage == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	newChall := Challenge{
		Image: dockerImage,
		Name:  containerName,
	}

	AddChall(newChall)

	chall, _ := GetAllChall()
	RenderTemplates(c, gin.H{
		"challenges": chall,
		"Message":    "Container Created!",
	}, "admin")

}

func removeContainerHandler(c *gin.Context) {
	containerName := c.PostForm("containerName")

	log.Println(containerName)

	// Validate input (you can add more validation logic)
	if containerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	RemoveChall(containerName)

	chall, _ := GetAllChall()
	RenderTemplates(c, gin.H{
		"challenges": chall,
		"Message":    "Container Deleted!",
	}, "admin")

}
