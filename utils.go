package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
)

type Challenge struct {
	Image   string
	Name    string
	Id      string
	Message string
	Type    string
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

func AddChall(chall Challenge) {
	challenges, err := GetAllChall()
	if err != nil {
		panic(err)
	}

	challenges = append(challenges, chall)

	challengesJson, err := json.Marshal(challenges)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("challenges.json", challengesJson, 0644)
	if err != nil {
		panic(err)
	}
}

func RemoveChall(challName string) {
	challenges, err := GetAllChall()
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(challenges); i++ {
		if challenges[i].Name == challName {
			challenges = append(challenges[:i], challenges[i+1:]...)
		}
	}

	challengesJson, err := json.Marshal(challenges)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("challenges.json", challengesJson, 0644)
	if err != nil {
		panic(err)
	}
}

func GetChallbyId(id string) Challenge {
	chall, err := GetAllChall()
	if err != nil {
		panic(err)
	}
	number_id, _ := strconv.Atoi(id)
	return chall[number_id]
}

func RenderTemplates(c *gin.Context, Data any, optionTemplateName ...string) {

	mainTemplateName := "main"

	if c.GetHeader("Hx-Request") == "true" {
		mainTemplateName = "htmx"
	}

	var templateName string

	if len(optionTemplateName) == 0 {
		templateName = c.Request.URL.Path

		if templateName == "/" {
			templateName = "main"
		}
	} else {
		templateName = optionTemplateName[0]
	}

	// 메인 템플릿 디렉토리
	mainTemplateDir := "templates/layouts/"

	// 템플릿 생성
	tmpl, err := template.New(mainTemplateName).ParseGlob(filepath.Join(mainTemplateDir, "*.tmpl"))
	if err != nil {
		return
	}

	// 서브 템플릿 등록
	subTemplatePath := filepath.Join("templates/pages/", templateName+".tmpl")
	_, err = tmpl.ParseFiles(subTemplatePath)
	if err != nil {
		return
	}

	// 렌더링 결과를 저장할 버퍼 생성
	var result bytes.Buffer

	// 템플릿 실행 및 결과를 버퍼에 쓰기
	err = tmpl.ExecuteTemplate(&result, mainTemplateName+".tmpl", Data)
	if err != nil {
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", result.Bytes())
}
