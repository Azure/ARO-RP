package location

import "strings"

// Normalize will take user friendly location name and return code normalized version
// Example: "East US 2" -> "eastus2"
func Normalize(location string) string {
	return strings.ReplaceAll(strings.ToLower(location), " ", "")
}
