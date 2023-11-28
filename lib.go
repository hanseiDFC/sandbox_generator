package main

import (
	"math/rand"
	"strconv"
	"time"
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
