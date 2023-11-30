package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Challenge struct {
	Image string
	Name  string
	Id    string
}

func GenerateId(data *gin.Context) string {
	hash := sha1.Sum([]byte(data.ClientIP() + data.Request.UserAgent() + time.Now().String()))
	return strings.ToLower(base64.RawURLEncoding.EncodeToString(hash[:])[:5])
}

func GetAllChall() ([]Challenge, error) {
	fileContent, err := os.ReadFile("challenges.json")
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON content into an array of Challenge structs
	var challenges []Challenge
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

func GetChallbyId(id string) Challenge {
	chall, err := GetAllChall()
	if err != nil {
		panic(err)
	}
	number_id, _ := strconv.Atoi(id)
	return chall[number_id]
}
