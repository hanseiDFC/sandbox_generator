package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
)

var used_ports []int
var online_sandbox_ids []string

func main() {

	router := gin.Default()

	router.GET("/", home)
	router.GET("/create", create)
	router.POST("/remove", remove)

	router.Run(":5000")
}

func remove_sandbox(id string) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{})

	fmt.Println("remove sandbox: " + id)
}

func sandbox_cleanup() {

	for _, online_sandbox_id := range online_sandbox_ids {
		remove_sandbox(online_sandbox_id)
	}
	online_sandbox_ids = []string{}
	used_ports = []int{}

	fmt.Println("All sandbox are removed!!")
}

func remove(c *gin.Context) {
	id := c.PostForm("id")
	// id := c.Query("id")
	return_msg := map[string]string{
		"massage":  "can't find sandbox",
		"received": id,
	}
	for _, online_sandbox_id := range online_sandbox_ids {
		if online_sandbox_id == id {
			remove_sandbox(id)
			return_msg["massage"] = "remove sandbox: " + id
		}
	}
	c.JSON(http.StatusOK, return_msg)
}

func create(c *gin.Context) {

	sandbox_port := random_port()
	user_name := "root"
	user_password := "root"

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	config := &container.Config{
		Image: "sshd",
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"22/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: sandbox_port,
				},
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	return_msg := map[string]string{
		"massage":  "success",
		"port":     sandbox_port,
		"user":     user_name,
		"password": user_password,
		"id":       resp.ID,
	}
	online_sandbox_ids = append(online_sandbox_ids, resp.ID)
	c.JSON(http.StatusOK, return_msg)
}

func random_port() string {
	rand.Seed(time.Now().UnixNano())

	port := rand.Intn(65535-1024) + 1024
	for _, arr := range used_ports {
		if port == arr {
			return random_port()
		}
	}
	used_ports = append(used_ports, port)
	return strconv.Itoa(port)
}

func home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Server Generation API for CTF",
	})
}
