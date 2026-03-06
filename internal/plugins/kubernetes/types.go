package kubernetes

// Minimal K8s API response types — only the fields Gantry needs.

type ObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	UID         string            `json:"uid"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// NamespaceList

type Namespace struct {
	Metadata ObjectMeta     `json:"metadata"`
	Status   NamespaceStatus `json:"status"`
}

type NamespaceStatus struct {
	Phase string `json:"phase"`
}

type NamespaceList struct {
	Items []Namespace `json:"items"`
}

// DeploymentList

type Deployment struct {
	Metadata ObjectMeta       `json:"metadata"`
	Spec     DeploymentSpec   `json:"spec"`
	Status   DeploymentStatus `json:"status"`
}

type DeploymentSpec struct {
	Replicas int32 `json:"replicas"`
}

type DeploymentStatus struct {
	Replicas      int32 `json:"replicas"`
	ReadyReplicas int32 `json:"readyReplicas"`
}

type DeploymentList struct {
	Items []Deployment `json:"items"`
}

// ServiceList (K8s Service, not Gantry Service)

type KService struct {
	Metadata ObjectMeta    `json:"metadata"`
	Spec     KServiceSpec  `json:"spec"`
}

type KServiceSpec struct {
	Type      string `json:"type"`      // ClusterIP | NodePort | LoadBalancer | ExternalName
	ClusterIP string `json:"clusterIP"`
}

type KServiceList struct {
	Items []KService `json:"items"`
}
