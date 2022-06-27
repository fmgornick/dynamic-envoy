package univcfg

const (
	INTERNAL = 0b01
	EXTERNAL = 0b10
	BOTH     = 0b11
)

type Config struct {
	Listeners map[string]*Listener   // should have one listener for internal, and one for external
	Clusters  map[string]*Cluster    // one cluster per domain, routes to 1+ endpoints
	Routes    map[string]*Route      // maps one url path to one cluster
	Endpoints map[string][]*Endpoint // one endpoint per upstream, clusters can map to 1+ endpoints
}

type Listener struct {
	Address string   // listen on a specific url
	Name    string   // either "internal" or "external"
	Port    uint     // should default to 443
	Routes  []string // maps to cluster from specific path
}

// if match.type not set to "exact" or "starts_with", then listener only has one route mapping to single cluster
type Route struct {
	Availability uint8  // tells us if the route is internal, external or both
	ClusterName  string // maps upstream from route, could have multiple upstreams
	Path         string // exact path must be specified
	Type         string // either "path" or "prefix"
}

type Cluster struct {
	Availability uint8  // tells us if the route is internal, external or both
	Name         string // should be the path of the url (or config id)
	Policy       string // load balancing policy, should default to round robin
}

type Endpoint struct {
	Address     string // where the user actually gets sent
	ClusterName string // name of cluster that owns the endpoint
	Port        uint   // default to 443
	Region      string // "global", "ttc", or "ttce"
	Weight      uint   // should default to 0 unless "Balance" set to weighted round robin
}

func NewConfig() *Config {
	return &Config{
		Listeners: make(map[string]*Listener),
		Clusters:  make(map[string]*Cluster),
		Routes:    make(map[string]*Route),
		Endpoints: make(map[string][]*Endpoint),
	}
}

func (cfg *Config) AddListener(address string, name string, port uint) {
	cfg.Listeners[name] = &Listener{
		Address: address,
		Name:    name,
		Port:    port,
	}
}

func (cfg *Config) AddCluster(name string, policy string) {
	var availability uint8
	switch name[len(name)-3:] {
	case "-in":
		availability = INTERNAL
	case "-ex":
		availability = EXTERNAL
	case "-ie":
		availability = BOTH
	default:
		panic("invalid availability")
	}
	cfg.Clusters[name] = &Cluster{
		Availability: availability,
		Name:         name,
		Policy:       policy,
	}
}

func (cfg *Config) AddRoute(clusterName string, path string, pathType string) {
	var availability uint8
	switch clusterName[len(clusterName)-3:] {
	case "-in":
		availability = INTERNAL
	case "-ex":
		availability = EXTERNAL
	case "-ie":
		availability = BOTH
	default:
		panic("invalid availability")
	}
	cfg.Routes[clusterName] = &Route{
		Availability: availability,
		ClusterName:  clusterName,
		Path:         path,
		Type:         pathType,
	}
}

func (cfg *Config) AddEndpoint(address string, clusterName string, port uint, region string, weight uint) {
	cfg.Endpoints[clusterName] = append(cfg.Endpoints[clusterName], &Endpoint{
		Address:     address,
		ClusterName: clusterName,
		Port:        port,
		Region:      region,
		Weight:      weight,
	})
}
