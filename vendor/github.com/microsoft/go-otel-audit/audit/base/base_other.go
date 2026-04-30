//go:build !unix

package base

func kernelVer() (string, error) {
	return "unsupported system, if you see this in logs, something is wrong", nil
}
