package topology

// The topology.json presentation
type Topology struct {
	// cluster name of this inspection
	ClusterName string `json:"cluster_name"`
	// the hosts of the cluster
	Hosts []struct {
		Ip         string `json:"ip"`
		Components []struct {
			// the name of compoennt, eg. tidb, tikv, pd
			Name string `json:"name"`
			// the port this component listen on
			Port string `json:"port"`
			// if this component alive
			Status string `json:"-"`
		} `json:"components"`
	} `json:"hosts"`
}
