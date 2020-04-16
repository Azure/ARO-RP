package testrandom

import (
	"math/rand"

	"github.com/onsi/ginkgo"
)

var rNumG = rand.New(rand.NewSource(ginkgo.GinkgoRandomSeed()))

// RandomInteger returns a random integer subject to the Ginkgo seed
func RandomInteger(max int) int {
	return rNumG.Intn(max)
}
