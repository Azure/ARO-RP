//go:generate go run ../../hack/assets

package deploy

import (
	"github.com/openshift/installer/data"
)

func init() {
	data.Assets = Assets
}
