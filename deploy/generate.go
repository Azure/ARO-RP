//go:generate go run ../hack/gendeploy -o rp-development.json
//go:generate go run ../hack/gendeploy -production -o rp-production.json
//go:generate go run ../hack/gendeploy -production -debug -o rp-production-debug.json

package deploy
