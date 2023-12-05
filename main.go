package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

var online_sandbox_ids []string

func main() {

	router := gin.Default()

	router.LoadHTMLGlob("templates/components/*")

	_, err := client.NewClientWithOpts()
	if err != nil {
		fmt.Println("Docker Client Error: ", err)
	}

	router.GET("/", func(c *gin.Context) {
		chall, _ := GetAllChall()

		RenderTemplates(c, gin.H{
			"challenges": chall,
		})
	})

	router.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		chall := GetChallbyId(id)

		RenderTemplates(c, chall, "challenge")
	})

	router.GET("/:id/new", create)
	router.GET("/:id/del", remove)

	env := os.Getenv("SAN_PORT")
	if env == "" {
		env = "8000"
	}

	log.Fatal(router.Run(":" + env))
}

func create(c *gin.Context) {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "docker client error - 1",
		})
		return
	}

	challenge_id := c.Param("id")

	host := strings.Split(c.Request.Host, ":")

	if len(host) == 1 {
		if c.Request.TLS != nil {
			// HTTPS인 경우 443번 포트로 설정
			host = append(host, "443")
		} else {
			// HTTP인 경우 80번 포트로 설정
			host = append(host, "80")
		}
	}

	referer := c.Request.Referer()

	if len(host) == 1 {
		if strings.Contains(referer, "https") {
			host = append(host, "443")
		} else {
			host = append(host, "80")
		}
	}

	// get hostname from url

	if challenge_id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "id is empty",
		})
		return
	}

	ctx := context.Background()

	chall := GetChallbyId(challenge_id)
	imageName := chall.Image

	fmt.Println("create sandbox: " + imageName)

	hashId := GenerateId(c)

	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		fmt.Println("pull image: " + imageName)
		out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "docker client error - fail to pull image",
			})
			return
		}

		// Wait for the image pull to complete
		var buf bytes.Buffer
		_, copyErr := io.Copy(&buf, out)
		if copyErr != nil {
			// Handle the copy error
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "docker client error - fail to read image pull output",
			})
			return
		}

		// Check if there are any errors reported in the output
		if strings.Contains(buf.String(), "error") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "docker client error - error in image pull output",
			})
			return
		}

		// Now the image pull is complete
		fmt.Println("Image pull complete for: " + imageName)
	}

	config := &container.Config{
		Image: imageName,
		Labels: map[string]string{
			"traefik.enable": "true",
			"traefik.tcp.routers." + hashId + ".rule": "HostSNI(`" + hashId + "." + host[0] + "`)",
			"traefik.tcp.routers." + hashId + ".tls":  "true",
			"dklodd":                                  "true",
		},
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "traefik",
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "docker client error - 2",
		})
		return
	}

	sandboxID := resp.ID

	// Start the container
	if err := cli.ContainerStart(ctx, sandboxID, types.ContainerStartOptions{}); err != nil {
		fmt.Println("Failed to start container:", err) // 에러 메시지 출력
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "docker client error - 3: failed to start container",
		})
		return
	}

	fmt.Println("create sandbox: " + sandboxID[0:12])

	online_sandbox_ids = append(online_sandbox_ids, sandboxID[0:12])

	c.HTML(http.StatusOK, "create.tmpl", gin.H{
		"Connection": gin.H{
			"ncat":    "ncat --ssl " + hashId + "." + host[0] + " " + host[1],
			"openssl": "openssl s_client -connect " + hashId + "." + host[0] + ":" + host[1],
		},
		"Id": sandboxID[0:12],
	})
}

func remove(c *gin.Context) {

	sandbox_id := c.Param("id")

	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	var message string

	for _, online_sandbox_id := range online_sandbox_ids {
		if online_sandbox_id == sandbox_id {
			if err := cli.ContainerStop(ctx, sandbox_id, nil); err != nil {
				message = "docker client error - 3: failed to stop container"
				break
			}

			if err := cli.ContainerRemove(ctx, sandbox_id, types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			}); err != nil {
				message = "docker client error - 4: failed to remove container"
				break
			}

			message = "scuccessfully removed sandbox"
			break
		}
	}

	if message == "" {
		message = "sandbox not found"
	}

	fmt.Println(message)

	c.HTML(http.StatusOK, "remove.tmpl", gin.H{
		"Message": message,
		"Id":      sandbox_id,
	})
}
