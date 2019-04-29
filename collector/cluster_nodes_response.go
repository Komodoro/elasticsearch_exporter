package collector

type clusterNodesResponse struct {
	ClusterName string `json:"cluster_name"`
	Nodes       map[string]ClusterNodesResponseNode
}

type ClusterNodesResponseNode struct {
	Name             string                         `json:"name"`
	EphemeralID      string                         `json:"ephemeral_id"`
	TransportAddress string                         `json:"transport_address"`
	Attributes       ClusterNodesResponseAttributes `json:"attributes"`
}

type ClusterNodesResponseAttributes struct {
	MlMaxOpenJobs string `json:"ml.max_open_jobs"`
	RackID        string `json:"rack_id"`
	MlEnabled     string `json:"ml.enabled"`
}
