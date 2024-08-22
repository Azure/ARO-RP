# Updating any of the constants in package `msgs`

If you update anything here, you need to run `go generate .` in this directory with the `stringer` tool installed.
You can even be more thorough by doing this from the top directory with `go generate ./...`

If you do not have stringer installed, you can install it with:

```bash
go install golang.org/x/tools/cmd/stringer@latest
```