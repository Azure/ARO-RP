package dummy

// Template represents a template
type Template struct {
	ID          string                 `json:"id,omitempty"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty"`
	Attachments string                 `json:"_attachments,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`
}

// Templates represent templates
type Templates struct {
	Count      int         `json:"_count,omitempty"`
	ResourceID string      `json:"_rid,omitempty"`
	Templates  []*Template `json:"Documents,omitempty"`
}
