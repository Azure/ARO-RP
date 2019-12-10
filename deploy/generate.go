//go:generate go run ../hack/gendeploy -development -o rp-development.json
//go:generate go run ../hack/gendeploy -o rp-production.json
//go:generate go run ../hack/gendeploy -debug -o rp-production-debug.json

package deploy
