package api

// OpenShiftClusterDocuments represents OpenShift cluster documents.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocuments struct {
	Count                     int                         `json:"_count,omitempty"`
	ResourceID                string                      `json:"_rid,omitempty"`
	OpenShiftClusterDocuments []*OpenShiftClusterDocument `json:"Documents,omitempty"`
}

// OpenShiftClusterDocument represents an OpenShift cluster document.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocument struct {
	MissingFields

	ID          string `json:"id,omitempty"`
	ResourceID  string `json:"_rid,omitempty"`
	Timestamp   int    `json:"_ts,omitempty"`
	Self        string `json:"_self,omitempty"`
	ETag        string `json:"_etag,omitempty"`
	Attachments string `json:"_attachments,omitempty"`

	SubscriptionID string `json:"subscriptionId,omitempty"` // partition key

	OpenShiftCluster *OpenShiftCluster `json:"openShiftCluster,omitempty"`

	Unqueued bool `json:"unqueued,omitempty"`
}
