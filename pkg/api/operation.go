package api

// OperationList represents an operation list.
type OperationList struct {
	// List of operations supported by the resource provider.
	Value []Operation `json:"value"`
}

// Operation represents an operation.
type Operation struct {
	// Operation name: {provider}/{resource}/{operation}.
	Name string `json:"name,omitempty"`

	// The object that describes the operation.
	Display Display `json:"display,omitempty"`
}

// Display represents the display details of an operation.
type Display struct {
	// Friendly name of the resource provider.
	Provider string `json:"provider,omitempty"`

	// Resource type on which the operation is performed.
	Resource string `json:"resource,omitempty"`

	// Operation type: read, write, delete, listKeys/action, etc.
	Operation string `json:"operation,omitempty"`

	// Friendly name of the operation.
	Description string `json:"description,omitempty"`
}
