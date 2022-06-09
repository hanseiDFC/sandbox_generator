from flask import Flask, jsonify, request
import atexit
import docker


app = Flask(__name__)
client = docker.from_env()

used_port = []
used_sandbox = []


def random_port():
    import random
    port = random.randint(9000, 49151)
    if port not in used_port:
        used_port.append(port)
        return port
    else:
        return random_port()


def remove_sandbox(sandbox_id):
    sandbox = client.containers.get(sandbox_id)
    sandbox.stop()
    sandbox.remove()
    print(f"{sandbox_id} is removed.")


@app.route('/')
def home():
    return jsonify({"massage": 'Server Generation API for CTF'})


@app.route('/create', methods=['GET'])
def create():
    sandbox_port = random_port()
    user_name = "root"
    user_password = "root"

    sandbox_id = client.containers.run(
        "sshd", detach=True, ports={22: sandbox_port}).id[:12]
    return_msg = {"massage": 'The sandbox is created.', "sandbox_port": sandbox_port,
                  "user_name": user_name, "sandbox_id": sandbox_id, "user_password": user_password}
    used_sandbox.append(return_msg)     # 이미 생성된 포트 및 프로그램 종료시 종료 작업 수행을 위해

    # 나중에 유저 토큰 추가
    return jsonify(return_msg)


@app.route('/remove', methods=['POST'])
def remove():
    id = request.json['id']
    for sandbox in used_sandbox:
        if sandbox['sandbox_id'] == id:
            remove_sandbox(sandbox['sandbox_id'])
            used_sandbox.remove(sandbox)
            return jsonify({"massage": 'The sandbox is removed.'})


if __name__ == '__main__':
    app.run(host='0.0.0.0', port='5000', debug=False)


@atexit.register
def sandbox_cleanup():
    for sandbox in used_sandbox:
        remove_sandbox(sandbox['sandbox_id'])
    print("All sandbox are removed.")
