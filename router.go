package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

type challenge struct {
	image string
	port  string
	id    int
}

var challenges = []challenge{
	{"minpeter/mathematician-in-wonderland", "5555", 1},
}

func get_image(id string) string {
	for _, challenge := range challenges {
		if id == strconv.Itoa(challenge.id) {
			return challenge.image
		}
	}
	return "not found"
}

func home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":    "Server Generation API for CTF 🚩🚩",
		"challenges": challenges,
	})
}

func create(c *gin.Context) {

	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	challenge_id := c.PostForm("id")

	ctx := context.Background()

	port := random_port()

	config := &container.Config{
		// TODO: add error handling
		Image: get_image(challenge_id),
		Labels: map[string]string{
			"traefik.enable":                                     "true",
			"traefik.http.routers." + port + ".rule":             "Host(`" + port + ".ctf.minpeter.tech`)",
			"traefik.http.routers." + port + ".entrypoints":      "websecure",
			"traefik.http.routers." + port + ".tls.certresolver": "myresolver",
		},
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "traefik",
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
		"massage": "success",
		// TODO: add load env from config file
		"url": "https://" + sandbox_id + ".ctf.minpeter.tech",
		"id":  sandbox_id,
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
