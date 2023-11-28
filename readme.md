## Dklodd for operating CTF system challenge with docker

TJCSec/klodd을 k8s 없이 운용할 수 있도록 만든 프로젝트입니다.
TJCsec과 아무런 연관이 없으며 단독적인 프로젝트임을 밝힙니다.

This is a project created to enable TJCSec/klodd to be operated without k8s.
We would like to clarify that this is an independent project and has no connection with TJCsec.

우아하게 설정된 단독 traefik와 하나의 머신 또는 VM에서 동작하도록 설계되었습니다.

기존의 nc 명령어 대신 ncat, socat, openssl를 사용하여 ssl 접속을 허용합니다.

### How to use

아직 실험적인 단계이므로, 사용에 주의가 필요합니다.

minpeter/homelab_infra 설정을 따른 후, docker-compose up -d 명령어를 통해 실행합니다.
