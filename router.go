package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Server Generation API for CTF",
	})
}

func create(c *gin.Context) {

	sandbox_port, user_name, user_password, sandbox_id := create_sandbox()

	return_msg := map[string]string{
		"massage":  "success",
		"port":     sandbox_port,
		"user":     user_name,
		"password": user_password,
		"id":       sandbox_id,
	}

	online_sandbox_ids = append(online_sandbox_ids, sandbox_id)

	c.JSON(http.StatusOK, return_msg)
}

func remove(c *gin.Context) {

	id := c.PostForm("id")

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
