package main

import (
	"github.com/jim-minter/rp/pkg/deploy"
)

func run() error {
	err := deploy.GenerateRPTemplates()
	if err != nil {
		return err
	}

	return deploy.GenerateNSGTemplate()
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
