package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func adminRouter(admin *gin.RouterGroup) {
	admin.GET("/", func(c *gin.Context) {
		chall, _ := GetAllChall()
		RenderTemplates(c, gin.H{
			"challenges": chall,
		}, "admin")
	})

	admin.POST("/image/add", createContainerHandler)

	admin.POST("/image/del", removeContainerHandler)

	admin.GET("/online", func(c *gin.Context) {
		t, _ := template.ParseFiles(filepath.Join("templates", "components", "online.tmpl"))
		t.Execute(c.Writer, GetOnlineSandbox())
	})

	admin.GET("/online/sse", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		c.Status(http.StatusOK)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				data := GetOnlineSandbox()
				var buf bytes.Buffer

				t, _ := template.ParseFiles(filepath.Join("templates", "components", "online.tmpl"))
				if err := t.Execute(&buf, data); err != nil {
					fmt.Println("Error executing template:", err)
					return
				}

				fmt.Fprintf(c.Writer, "data: %s\n\n", strings.ReplaceAll(buf.String(), "\n", ""))
				c.Writer.Flush()

			case <-c.Writer.CloseNotify():
				return
			}
		}
	})
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