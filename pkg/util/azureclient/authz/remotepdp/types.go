package remotepdp

type AuthorizationRequest struct {
	Subject            SubjectInfo     `json:"Subject"`
	Actions            []ActionInfo    `json:"Actions"`
	Resource           ResourceInfo    `json:"Resource"`
	Environment        EnvironmentInfo `json:"Environment,omitempty"`
	CheckClassicAdmins bool            `json:"CheckClassicAdmins,omitempty"`
}

type SubjectInfo struct {
	Attributes SubjectAttributes `json:"Attributes"`
}

type SubjectAttributes struct {
	ObjectId         string   `json:"ObjectId"`
	Groups           []string `json:"Groups"`
	ApplicationId    string   `json:"ApplicationId,omitempty"`
	ApplicationACR   string   `json:"ApplicationACR,omitempty"`
	RoleTemplate     []string `json:"RoleTemplate,omitempty"`
	TenantId         string   `json:"tid,omitempty"`
	Scope            string   `json:"Scope,omitempty"`
	ResourceId       string   `json:"ResourceId,omitempty"`
	Puid             string   `json:"puid,omitempty"`
	AltSecId         string   `json:"altsecid,omitempty"`
	IdentityProvider string   `json:"idp,omitempty"`
	Issuer           string   `json:"iss,omitempty"`
}

type ActionInfo struct {
	Id           string `json:"Id"`
	IsDataAction bool   `json:"IsDataAction,omitempty"`
	Attributes   `json:"Attributes"`
}

type ResourceInfo struct {
	Id         string `json:"Id"`
	Attributes `json:"Attributes"`
}

type EnvironmentInfo struct {
	Attributes `json:"Attributes"`
}

type AuthorizationDecisionResponse struct {
	Value    []AuthorizationDecision `json:"value"`
	NextLink string                  `json:"nextLink"`
}

type AuthorizationDecision struct {
	ActionId       string `json:"actionId,omitempty"`
	AccessDecision `json:"accessDecision,omitempty"`
	IsDataAction   bool `json:"isDataAction,omitempty"`
	RoleAssignment `json:"roleAssignment,omitempty"`
	DenyAssignment RoleDefinition `json:"denyAssignment,omitempty"`
	TimeToLiveInMs int            `json:"timeToLiveInMs,omitempty"`
}

type AccessDecision string

type RoleAssignment struct {
	Id                                 string `json:"id,omitempty"`
	RoleDefinitionId                   string `json:"roleDefinitionId,omitempty"`
	PrincipalId                        string `json:"principalId,omitempty"`
	PrincipalType                      string `json:"principaltype,omitempty"`
	Scope                              string `json:"scope,omitempty"`
	Condition                          string `json:"condition,omitempty"`
	ConditionVersion                   string `json:"conditionVersion,omitempty"`
	CanDelegate                        bool   `json:"canDelegate,omitempty"`
	DelegatedManagedIdentityResourceId string `json:"deletegatedManagedIdentityResourceId,omitempty"`
	Description                        string `json:"description,omitempty"`
}

type RoleDefinition struct {
	Id string `json:"id,omitempty"`
}

//
type Attributes map[string]interface{}

// RemotePDPErrorPayload represents the body content when the server returns
// a non-successful error
type CheckAccessErrorResponse struct {
	StatusCode string `json:"statusCode,omitempty"`
	Message    string `json:"message,omitempty"`
}
