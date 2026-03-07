package argocd

// Minimal ArgoCD API response types — only the fields Gantry needs.

// ApplicationList is the response from GET /api/v1/applications.
type ApplicationList struct {
	Items []Application `json:"items"`
}

// Application represents a single ArgoCD Application resource.
type Application struct {
	Metadata AppMeta `json:"metadata"`
	Spec     AppSpec `json:"spec"`
	Status   AppStatus `json:"status"`
}

// AppMeta contains basic Kubernetes object metadata.
type AppMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	UID       string            `json:"uid"`
	Labels    map[string]string `json:"labels"`
}

// AppSpec describes the desired state of an ArgoCD Application.
type AppSpec struct {
	Source      AppSource      `json:"source"`
	Destination AppDestination `json:"destination"`
	Project     string         `json:"project"`
}

// AppSource is the Git/Helm source for the application.
type AppSource struct {
	RepoURL        string `json:"repoURL"`
	TargetRevision string `json:"targetRevision"`
	Path           string `json:"path,omitempty"`
	Chart          string `json:"chart,omitempty"`
}

// AppDestination is the target cluster and namespace.
type AppDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
	Name      string `json:"name,omitempty"` // cluster name alias
}

// AppStatus reflects the live state of an ArgoCD Application.
type AppStatus struct {
	Sync           SyncStatus       `json:"sync"`
	Health         HealthStatus     `json:"health"`
	OperationState *OperationState  `json:"operationState,omitempty"`
	Summary        AppSummary       `json:"summary"`
	Resources      []ResourceStatus `json:"resources,omitempty"`
}

// SyncStatus describes whether the app is in sync with its source.
type SyncStatus struct {
	Status   string `json:"status"` // Synced | OutOfSync | Unknown
	Revision string `json:"revision,omitempty"`
}

// HealthStatus describes the health of an ArgoCD Application.
type HealthStatus struct {
	Status  string `json:"status"`  // Healthy | Degraded | Progressing | Missing | Unknown | Suspended
	Message string `json:"message,omitempty"`
}

// OperationState carries the result of the last sync operation.
type OperationState struct {
	Phase   string `json:"phase,omitempty"`   // Running | Succeeded | Error | Failed
	Message string `json:"message,omitempty"`
}

// AppSummary carries high-level metadata about the app's content.
type AppSummary struct {
	Images []string `json:"images,omitempty"`
}

// ResourceStatus represents a managed Kubernetes resource.
type ResourceStatus struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	Health    *HealthStatus `json:"health,omitempty"`
}

// AppWithInstance bundles an AppStatusResponse with the name of the ArgoCD
// instance it came from, so the frontend can display per-instance info.
type AppWithInstance struct {
	Instance string `json:"instance"`
	*AppStatusResponse
}

// AppStatusResponse is the live status returned by the Gantry API.
type AppStatusResponse struct {
	AppName        string           `json:"appName"`
	SyncStatus     string           `json:"syncStatus"`
	HealthStatus   string           `json:"healthStatus"`
	HealthMessage  string           `json:"healthMessage,omitempty"`
	SyncRevision   string           `json:"syncRevision,omitempty"`
	OperationPhase string           `json:"operationPhase,omitempty"`
	OperationMsg   string           `json:"operationMsg,omitempty"`
	RepoURL        string           `json:"repoURL,omitempty"`
	TargetRevision string           `json:"targetRevision,omitempty"`
	Path           string           `json:"path,omitempty"`
	Chart          string           `json:"chart,omitempty"`
	Project        string           `json:"project,omitempty"`
	DestServer     string           `json:"destServer,omitempty"`
	DestNamespace  string           `json:"destNamespace,omitempty"`
	Images         []string         `json:"images,omitempty"`
	Resources      []ResourceStatus `json:"resources,omitempty"`
}
