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
        echo "Killing Podman process with PID: $PODMAN_PID"
        kill $PODMAN_PID 2>/dev/null
        if [ $? -ne 0 ]; then
            echo "Error: Failed to kill Podman process with PID $PODMAN_PID."
            exit 1
        fi
        wait $PODMAN_PID 2>/dev/null || echo "Podman process $PODMAN_PID was not a child of this shell."
    else
        echo "No PODMAN_PID found. Attempting to kill by port."
        rppid=$(lsof -t -i :8888)
        if [ -n "$rppid" ]; then
            echo "Killing Podman running on port 8888 with PID $rppid."
            kill $rppid
            if [ $? -ne 0 ]; then
                echo "Error: Failed to kill Podman running on port 8888."
                exit 1
            fi
        else
            echo "No process found running on port 8888."
        fi
    fi
}

