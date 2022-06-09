package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

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

func remove_sandbox(sandbox_id string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

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
}

func create_sandbox() (string, string, string, string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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

	return sandbox_port, user_name, user_password, sandbox_id
}

// func sandbox_cleanup() {

// 	for _, online_sandbox_id := range online_sandbox_ids {
// 		remove_sandbox(online_sandbox_id)
// 	}
// 	online_sandbox_ids = []string{}
// 	used_ports = []int{}

// 	fmt.Println("All sandbox are removed!!")
// }
