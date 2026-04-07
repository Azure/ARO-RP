## Install Go via `gvm`

1. Install dependencies
    ```sh
    sudo dnf install -y \
        bison
    ```
2. [Install `gvm`](https://github.com/moovweb/gvm?tab=readme-ov-file#installing)
    ```sh
    bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
    ```
3. [Install Go](https://github.com/moovweb/gvm?tab=readme-ov-file#installing-go)
    ```sh
    gvm install go1.25.3
    ```
4. View other `gvm` options
    ```sh
    gvm help
    ```