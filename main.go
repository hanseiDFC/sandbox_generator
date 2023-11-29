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

	_, err := client.NewClientWithOpts()
	if err != nil {
		fmt.Println("Docker Client Error: ", err)
	}

	router.GET("/", home)
	router.GET("/new/:id", create)
	router.GET("/del/:id", remove)

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

func get_image(id string) string {

	chall, err := load_challenges()
	if err != nil {
		panic(err)
	}
	number_id, _ := strconv.Atoi(id)
	return chall[number_id].Image
}

func home(c *gin.Context) {

	chall, _ := load_challenges()

	host := c.Request.Host

	c.JSON(http.StatusOK, gin.H{
		"message":    "Server Generation API for CTF 🚩🚩",
		"challenges": chall,
		"Host":       host,
		"schema":     c.Request.URL.Scheme,
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
		if c.Request.URL.Scheme == "https" {
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

	imageName := get_image(challenge_id)

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
		Image: get_image(challenge_id),
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

	c.JSON(http.StatusOK,
		gin.H{
			"url":  port + "." + host[0],
			"port": host[1],
			"id":   sandboxID[0:12],
			"connection": gin.H{
				"ncat":    "ncat --ssl " + port + "." + host[0] + " " + host[1],
				"openssl": "openssl s_client -connect " + port + "." + host[0] + ":" + host[1],
			},
		},
	)
}

func remove(c *gin.Context) {

	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	sandbox_id := c.Param("id")

	return_msg := map[string]string{
		"received": sandbox_id,
	}

	for _, online_sandbox_id := range online_sandbox_ids {
		if online_sandbox_id == sandbox_id {
			if err := cli.ContainerStop(ctx, sandbox_id, nil); err != nil {
				panic(err)
			}

			removeOptions := types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			}

			if err := cli.ContainerRemove(ctx, sandbox_id, removeOptions); err != nil {
				panic(err)
			}

			fmt.Println("remove sandbox: " + sandbox_id)

			return_msg["massage"] = "remove sandbox: " + sandbox_id

			break

		} else {
			return_msg["massage"] = "can't find sandbox"
		}
	}

	c.JSON(http.StatusOK, return_msg)
}
