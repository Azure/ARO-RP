//go:generate go run ../../hack/assets

package install

import (
	"github.com/openshift/installer/data"
)

func init() {
	data.Assets = Assets
}
