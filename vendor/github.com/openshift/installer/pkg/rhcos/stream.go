//go:build !okd
// +build !okd

package rhcos

func getStreamFileName() string {
	return "coreos/rhcos.json"
}
