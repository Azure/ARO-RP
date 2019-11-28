package main

import (
	"flag"

	"github.com/jim-minter/rp/pkg/swagger"
)

var (
	outputFile   = flag.String("o", "", "output file")
	inputVersion = flag.String("i", "", "api version for input [example v20190211preview]")
)

func main() {
	flag.Parse()

	err := swagger.ValidateVersion(*inputVersion)
	if err != nil {
		panic(err)
	}

	if err := swagger.Run(*outputFile, *inputVersion); err != nil {
		panic(err)
	}
}
