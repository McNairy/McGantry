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
	Metadata ObjectMeta   `json:"metadata"`
	Spec     KServiceSpec `json:"spec"`
}

type KServiceSpec struct {
	Type      string            `json:"type"`      // ClusterIP | NodePort | LoadBalancer | ExternalName
	ClusterIP string            `json:"clusterIP"`
	Selector  map[string]string `json:"selector,omitempty"`
}

type KServiceList struct {
	Items []KService `json:"items"`
}

// DeploymentStatus — extended with AvailableReplicas.

// PodList

type Pod struct {
	Metadata ObjectMeta `json:"metadata"`
	Spec     PodSpec    `json:"spec"`
	Status   PodStatus  `json:"status"`
}

type PodSpec struct {
	NodeName   string         `json:"nodeName"`
	Containers []PodContainer `json:"containers"`
}

type PodContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type PodStatus struct {
	Phase             string            `json:"phase"`
	StartTime         string            `json:"startTime,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses"`
}

type ContainerStatus struct {
	Name         string         `json:"name"`
	Ready        bool           `json:"ready"`
	RestartCount int32          `json:"restartCount"`
	State        ContainerState `json:"state"`
}

type ContainerState struct {
	Running    *ContainerStateRunning    `json:"running,omitempty"`
	Waiting    *ContainerStateWaiting    `json:"waiting,omitempty"`
	Terminated *ContainerStateTerminated `json:"terminated,omitempty"`
}

type ContainerStateRunning struct{}

type ContainerStateWaiting struct {
	Reason string `json:"reason"`
}

type ContainerStateTerminated struct {
	Reason   string `json:"reason"`
	ExitCode int32  `json:"exitCode"`
}

type PodList struct {
	Items []Pod `json:"items"`
}

// WorkloadInfo is returned by the Kubernetes workload API endpoint.

type WorkloadInfo struct {
	AppName     string           `json:"appName"`
	Deployments []DeploymentInfo `json:"deployments"`
	Pods        []PodInfo        `json:"pods"`
}

type DeploymentInfo struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	DesiredReplicas   int32  `json:"desiredReplicas"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
}

type PodInfo struct {
	Name          string          `json:"name"`
	Namespace     string          `json:"namespace"`
	Phase         string          `json:"phase"`
	Ready         bool            `json:"ready"`
	TotalRestarts int32           `json:"totalRestarts"`
	NodeName      string          `json:"nodeName,omitempty"`
	StartTime     string          `json:"startTime,omitempty"`
	Containers    []ContainerInfo `json:"containers"`
}

type ContainerInfo struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Ready    bool   `json:"ready"`
	Restarts int32  `json:"restarts"`
	State    string `json:"state"` // running | waiting | terminated | unknown
	Reason   string `json:"reason,omitempty"`
}
