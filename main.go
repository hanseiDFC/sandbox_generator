package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

var used_ports []int
var online_sandbox_ids []string

func main() {

	router := gin.Default()

	router.LoadHTMLGlob("templates/*")

	_, err := client.NewClientWithOpts()
	if err != nil {
		fmt.Println("Docker Client Error: ", err)
	}

	router.GET("/", func(c *gin.Context) {

		chall, _ := load_challenges()
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"challenges": chall,
		})
	})

	router.GET("/:id", func(c *gin.Context) {

		id := c.Param("id")

		chall := get_chall(id)

		c.HTML(http.StatusOK, "challenge.tmpl", chall)
	})

	router.GET("/:id/new", create)
	router.GET("/:id/del", remove)

	// 환경변수에 SAN_PORT가 있으면 이용 없으면 5000

	env := os.Getenv("SAN_PORT")
	if env == "" {
		env = "5000"
	}

	router.Run(":" + env)
}

func random_port() string {
	rand.Seed(time.Now().UnixNano())

	port := rand.Intn(65535-1024) + 1024

	for _, rpn := range used_ports { // Random Port Number
		if port == rpn {
			return random_port()
		}
	}
	used_ports = append(used_ports, port)
	return strconv.Itoa(port)
}

type challenge struct {
	Image string
	Name  string
	Id    string
}

func load_challenges() ([]challenge, error) {
	// Read the content of the JSON file
	fileContent, err := ioutil.ReadFile("challenges.json")
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON content into an array of Challenge structs
	var challenges []challenge
	err = json.Unmarshal(fileContent, &challenges)
	if err != nil {
		return nil, err
	}

	var ChallengeId int
	for i := 0; i < len(challenges); i++ {
		ChallengeId = i
		challenges[i].Id = strconv.Itoa(ChallengeId)
	}

	return challenges, nil
}

func get_chall(id string) challenge {

	chall, err := load_challenges()
	if err != nil {
		panic(err)
	}
	number_id, _ := strconv.Atoi(id)
	return chall[number_id]
}

func home(c *gin.Context) {

	chall, _ := load_challenges()

	host := c.Request.Host

	c.JSON(http.StatusOK, gin.H{
		"message":    "Server Generation API for CTF 🚩🚩",
		"challenges": chall,
		"Host":       host,
	})
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

	chall := get_chall(challenge_id)
	imageName := chall.Image

	fmt.Println("create sandbox: " + imageName)

	port := random_port()

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
		// TODO: add error handling
		Image: imageName,
		Labels: map[string]string{
			"traefik.enable":                        "true",
			"traefik.tcp.routers." + port + ".rule": "HostSNI(`" + port + "." + host[0] + "`)",
			"traefik.tcp.routers." + port + ".tls":  "true",
			"dklodd":                                "true",
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "docker client error - 3: failed to start container",
		})
		return
	}

	fmt.Println("create sandbox: " + sandboxID[0:12])

	online_sandbox_ids = append(online_sandbox_ids, sandboxID[0:12])

	c.HTML(http.StatusOK, "create.tmpl", gin.H{
		"Connection": gin.H{
			"ncat":    "ncat --ssl " + port + "." + host[0] + " " + host[1],
			"openssl": "openssl s_client -connect " + port + "." + host[0] + ":" + host[1],
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
