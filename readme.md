## build sshd docker images
`docker build -f dockerfile_sshd -t sshd .`  
** 추후에 문제 이미지 파일 업로드 로직을 따로 개발할 예정  

## start api server
`go run *.go`  