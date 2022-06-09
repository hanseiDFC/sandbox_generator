package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
)

func home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Server Generation API for CTF 🚩🚩",
	})
}

func create(c *gin.Context) {

	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	sandbox_port := random_port()
	user_name := "root"
	user_password := "root"

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

	sandbox_id := resp.ID

	if err := cli.ContainerStart(ctx, sandbox_id, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Println("create sandbox: " + sandbox_id)

	online_sandbox_ids = append(online_sandbox_ids, sandbox_id)

	return_msg := map[string]string{
		"massage":  "success",
		"port":     sandbox_port,
		"user":     user_name,
		"password": user_password,
		"id":       sandbox_id,
	}

	c.JSON(http.StatusOK, return_msg)
}

func remove(c *gin.Context) {

	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	sandbox_id := c.PostForm("id")

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
