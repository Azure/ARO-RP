package main

import (
	"flag"

	"github.com/jim-minter/rp/pkg/swagger"
)

var (
	outputFile = flag.String("o", "", "output file")
)

func main() {
	flag.Parse()

	if err := swagger.Run(*outputFile); err != nil {
		panic(err)
	}
}
