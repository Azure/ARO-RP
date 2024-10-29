#!/bin/bash -e

run_podman() {
    echo "########## 🚀 Run Podman in background ##########"
    podman --log-level=debug system service --time=0 tcp://127.0.0.1:8888 > podmanlog 2>&1 &
    PODMAN_PID=$!
    echo "Podman PID: $PODMAN_PID"
}

validate_podman_running() {
    echo "########## ？Checking podman Status ##########"
    ELAPSED=0
    while true; do
        sleep 5
        http_code=$(curl -k -s -o /dev/null -w '%{http_code}' http://localhost:8888/v1.30/_ping || true)
        case $http_code in
        "200")
            echo "########## ✅ Podman Running ##########"
            break
            ;;
        *)
            echo "Attempt $ELAPSED - podman is NOT up. Code : $http_code, waiting"
            sleep 2
            # after 40 secs return exit 1 to not block ci
            ELAPSED=$((ELAPSED + 1))
            if [ $ELAPSED -eq 20 ]; then
                echo "########## ❌ Podman failed to start within timeout ##########"
                kill_podman
                exit 1
            fi
            ;;
        esac
    done
}

kill_podman() {
    echo "podman logs:"
    cat podmanlog
    echo "########## Kill the podman running in background ##########"
    
    if [ -n "$PODMAN_PID" ]; then
        kill $PODMAN_PID 2>/dev/null
        wait $PODMAN_PID 2>/dev/null || echo "Podman process $PODMAN_PID was not a child of this shell."
    else
        echo "No PODMAN_PID found. Attempting to kill by port."
        rppid=$(lsof -t -i :8888)
        if [ -n "$rppid" ]; then
            kill $rppid
            echo "Killed podman running on port 8888 with PID $rppid."
        else
            echo "No process found running on port 8888."
        fi
    fi
}

setup_environment() {
    echo "########## 🌐 Setting up Azure account and secrets ##########"
    az account set -s "$AZURE_SUBSCRIPTION_ID"
    SECRET_SA_ACCOUNT_NAME="$SECRET_SA_ACCOUNT_NAME" make secrets
    . secrets/env
    export CI=true
    echo "Environment setup complete."
}

install_docker_dependencies() {
    echo "########## 🐳 Installing Docker and Docker Compose Plugin ##########"
    sudo apt-get update
    sudo apt-get install -y ca-certificates curl gnupg
    sudo install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo tee /etc/apt/keyrings/docker.asc
    sudo chmod a+r /etc/apt/keyrings/docker.asc
    echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
    $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin make
    sudo systemctl start docker
    sudo systemctl enable docker
    docker compose version
    echo "Docker and dependencies installed successfully."
}