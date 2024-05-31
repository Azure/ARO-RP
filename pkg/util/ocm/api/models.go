package api

import "time"

// UpgradePolicyList represents an unmarshalled Upgrade Policy response from Cluster Services
type UpgradePolicyList struct {
	Kind  string          `json:"kind"`
	Page  int64           `json:"page"`
	Size  int64           `json:"size"`
	Total int64           `json:"total"`
	Items []UpgradePolicy `json:"items"`
}

// UpgradePolicyStatus represents an unmarshalled Upgrade Policy Status response from Cluster Services
type UpgradePolicyStatus struct {
	State       string `json:"value"`
	Description string `json:"description"`
}

// UpgradePolicy represents an unmarshalled individual Upgrade Policy response from Cluster Services
type UpgradePolicy struct {
	Id                  string `json:"id"`
	Kind                string `json:"kind"`
	Href                string `json:"href"`
	Schedule            string `json:"schedule"`
	ScheduleType        string `json:"schedule_type"`
	UpgradeType         string `json:"upgrade_type"`
	Version             string `json:"version"`
	NextRun             string `json:"next_run"`
	PrevRun             string `json:"prev_run"`
	ClusterId           string `json:"cluster_id"`
	CapacityReservation *bool  `json:"capacity_reservation"`
	UpgradePolicyStatus
}

// ClusterList represents an unmarshalled Cluster List response from Cluster Services
type ClusterList struct {
	Kind  string        `json:"kind"`
	Page  int64         `json:"page"`
	Size  int64         `json:"size"`
	Total int64         `json:"total"`
	Items []ClusterInfo `json:"items"`
}

// ClusterInfo represents a partial unmarshalled Cluster response from Cluster Services
type ClusterInfo struct {
	Id                   string               `json:"id"`
	Name                 string               `json:"name"`
	ExternalID           string               `json:"external_id"`
	DisplayName          string               `json:"display_name"`
	CreationTimestamp    time.Time            `json:"creation_timestamp"`
	ActivityTimestamp    time.Time            `json:"activity_timestamp"`
	OpenshiftVersion     string               `json:"openshift_version"`
	Version              ClusterVersion       `json:"version"`
	NodeDrainGracePeriod NodeDrainGracePeriod `json:"node_drain_grace_period"`
	UpgradePolicies      []UpgradePolicy      `json:"upgrade_policies"`
}

// NodeDrainGracePeriod represents a duration for node drain grace periods
type NodeDrainGracePeriod struct {
	Value int64  `json:"value"`
	Unit  string `json:"unit"`
}

// ClusterVersion represents a clusters version
type ClusterVersion struct {
	Id                 string    `json:"id"`
	ChannelGroup       string    `json:"channel_group"`
	AvailableUpgrades  []string  `json:"available_upgrades"`
	EndOfLifeTimestamp time.Time `json:"end_of_life_timestamp"`
}

// UpgradePolicyStateRequest represents an Upgrade Policy state for notifications
type UpgradePolicyStateRequest struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

// UpgradePolicyState represents an Upgrade Policy state for notifications
type UpgradePolicyState struct {
	Kind string `json:"kind"`
	Href string `json:"href"`
	UpgradePolicyStatus
}

// Config represents the configmap data for the managed-upgrade-operator
type Config struct {
	ConfigManager ConfigManager `yaml:"configManager"`
}

// ConfigManager represents the config manager data for the managed-upgrade-operator
type ConfigManager struct {
	Source     string `yaml:"source"`
	OcmBaseURL string `yaml:"ocmBaseUrl"`
}

// Error represents an error response from the API server
type Error struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Href        string `json:"href"`
	Code        string `json:"code"`
	Reason      string `json:"reason"`
	OperationID string `json:"operation_id"`
}

// CancelUpgradeResponse represents a response from the API server
type CancelUpgradeResponse struct {
	Kind        string `json:"kind"`
	Href        string `json:"href"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
