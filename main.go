package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

var online_sandbox_ids []string

func GetOnlineSandbox() []Challenge {

	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	var resp []Challenge
	for i, online_sandbox_id := range online_sandbox_ids {
		data, err := cli.ContainerInspect(context.Background(), online_sandbox_id)
		if err != nil {
			fmt.Println("Failed to inspect container:", err) // 에러 메시지 출력
			online_sandbox_ids = append(online_sandbox_ids[:i], online_sandbox_ids[i+1:]...)
			continue
		}

		resp = append(resp, Challenge{
			Id:      data.ID[0:12],
			Name:    data.Config.Image,
			Message: data.State.Status,
		})

	}

	return resp

}

func ResetSandbox() {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	for _, online_sandbox_id := range online_sandbox_ids {
		if err := cli.ContainerStop(ctx, online_sandbox_id, nil); err != nil {
			fmt.Println("Failed to stop container:", err) // 에러 메시지 출력
			continue
		}

		if err := cli.ContainerRemove(ctx, online_sandbox_id, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			fmt.Println("Failed to remove container:", err) // 에러 메시지 출력
			continue
		}
	}

	online_sandbox_ids = nil

}

func LoadOnlineSandbox() {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		if container.Labels["dklodd"] == "true" {
			online_sandbox_ids = append(online_sandbox_ids, container.ID[0:12])
		}
	}
}

func main() {

	router := gin.Default()
	LoadOnlineSandbox()

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

	router.Static("/assets", "templates/assets")

	router.GET("/:id/new", create)
	router.GET("/:id/del", remove)

	// 어드민 전용 라우터 생성
	admin := router.Group("/admin", gin.BasicAuth(gin.Accounts{
		"admin": "admin",
	}))

	adminRouter(admin)

	env := os.Getenv("PORT")
	if env == "" {
		env = "8000"
	}

	host := ":" + env
	// Removes the “accept incoming network connections?” pop-up on Macos.
	if runtime.GOOS == "darwin" {
		host = "localhost:" + env
	}

	log.Fatal(router.Run(host))
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

	if chall.Type == "web" {
		config.Labels = map[string]string{
			"traefik.enable": "true",
			"traefik.http.routers." + hashId + ".rule": "Host(`" + hashId + "." + host[0] + "`)",
			"traefik.http.routers." + hashId + ".tls":  "true",
			"dklodd": "true",
		}
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

	if chall.Type == "web" {

		connection := "https://" + hashId + "." + host[0]

		if host[1] != "443" {
			connection += ":" + host[1]
		}

		c.HTML(http.StatusOK, "web.tmpl", gin.H{
			"Connection": connection,
			"Id":         sandboxID[0:12],
		})
	} else {
		c.HTML(http.StatusOK, "tcp.tmpl", gin.H{
			"Connection": gin.H{
				"ncat":    "ncat --ssl " + hashId + "." + host[0] + " " + host[1],
				"openssl": "openssl s_client -connect " + hashId + "." + host[0] + ":" + host[1],
			},
			"Id": sandboxID[0:12],
		})
	}
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

			for i, online_sandbox_id := range online_sandbox_ids {
				if online_sandbox_id == sandbox_id {
					online_sandbox_ids = append(online_sandbox_ids[:i], online_sandbox_ids[i+1:]...)
				}
			}

			message = "scuccessfully removed sandbox"
			break
		}
	}

	if message == "" {
		message = "sandbox not found"
	}

	fmt.Println(message)

	id := c.GetHeader("HX-Current-URL")
	id = strings.Split(id, "/")[len(strings.Split(id, "/"))-1]

	chall := GetChallbyId(id)

	chall.Message = message + " - " + sandbox_id

	RenderTemplates(c, chall, "challenge")
}
