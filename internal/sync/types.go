package sync

type App struct {
	ResourceID     string   `json:"resourceId"`
	Name           string   `json:"name"`
	SpaceID        string   `json:"spaceId"`
	SpaceName      string   `json:"spaceName"`
	SpaceType      string   `json:"spaceType"`
	AppType        string   `json:"appType"`
	OwnerID        string   `json:"ownerId"`
	OwnerName      string   `json:"ownerName"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	Published      bool     `json:"published"`
	LastReloadTime string   `json:"lastReloadTime"`
	Tenant         string   `json:"tenant"`
	TenantID       string   `json:"tenantId"`
	TargetPath     string   `json:"targetPath"`
	Skip           bool     `json:"skip"`
	SkipReason     string   `json:"skipReason"`
}

type Filters struct {
	Space  string
	Stream string
	App    string
	ID     string
}

type Result struct {
	ResourceID string `json:"resourceId"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type PrepOutput struct {
	Tenant   string `json:"tenant"`
	TenantID string `json:"tenantId"`
	Context  string `json:"context"`
	Server   string `json:"server"`
	Apps     []App  `json:"apps"`
}

type IndexEntry struct {
	Name           string   `json:"name"`
	Space          string   `json:"space"`
	SpaceID        string   `json:"spaceId"`
	SpaceType      string   `json:"spaceType"`
	AppType        string   `json:"appType"`
	Owner          string   `json:"owner"`
	OwnerName      string   `json:"ownerName"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	Published      bool     `json:"published"`
	LastReloadTime string   `json:"lastReloadTime"`
	Path           string   `json:"path"`
}

type Index struct {
	LastSync string                `json:"lastSync"`
	Context  string                `json:"context"`
	Server   string                `json:"server"`
	Tenant   string                `json:"tenant"`
	TenantID string                `json:"tenantId"`
	AppCount int                   `json:"appCount"`
	Apps     map[string]IndexEntry `json:"apps"`
}
