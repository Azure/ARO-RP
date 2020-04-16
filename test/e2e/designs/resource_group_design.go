package designs

const (
	RG_VNET = "VNET"
	RG_AKS  = "AKS"
)

// ResourceGroup represents the users intention for a resource group
type ResourceGroup struct {
	Name     string
	Location string
	Type     string
}
