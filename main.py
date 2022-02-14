from flask import Flask, jsonify
import docker


app = Flask(__name__)


@app.route('/')
def home():
    return 'This api is the api that creates the sandbox of hansei wargame.'


@app.route('/create')
def create():
    sandbox_port = 23456
    user_name = "root"
    user_password = "root"
    client = docker.from_env()
    client.containers.run("sshd", detach=True, ports={22: sandbox_port})
    return jsonify({"name": user_name, "password": user_password, "port": sandbox_port})


if __name__ == '__main__':
    app.run(host='0.0.0.0', port='5000', debug=True)
