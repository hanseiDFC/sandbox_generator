# Makefile

export COMPOSE_FILE := infra-config/dev-compose.yml

# 개발 서버를 실행하고 air 명령을 실행
dev:
		docker-compose -f $(COMPOSE_FILE) up -d
		air

# 데브 인프라를 완전히 종료
down:
		if [ -n "$(docker ps -qaf "label=dklodd")" ]; then \
			docker rm -f $(docker ps -qaf "label=dklodd"); \
		else \
			echo "No containers to remove"; \
		fi
		docker-compose -f $(COMPOSE_FILE) down

restart:
		docker-compose -f $(COMPOSE_FILE) restart
		air

# Go 빌드 실행
build:
		go build -o dklodd
