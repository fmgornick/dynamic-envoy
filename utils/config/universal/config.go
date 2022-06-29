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

// initialize map fields in our object
func NewConfig() *Config {
	return &Config{
		Listeners: make(map[string]*Listener),
		Clusters:  make(map[string]*Cluster),
		Routes:    make(map[string]*Route),
		Endpoints: make(map[string][]*Endpoint),
	}
}

// add a listener to our configuration object
func (cfg *Config) AddListener(address string, name string, port uint) {
	cfg.Listeners[name] = &Listener{
		Address: address,
		Name:    name,
		Port:    port,
	}
}

// add a cluster to our configuration object
// also set availability flag based on cluster name
func (cfg *Config) AddCluster(name string, policy string) {
	var availability uint8
	switch name[len(name)-2:] {
	case "in":
		availability = INTERNAL
	case "ex":
		availability = EXTERNAL
	case "ie":
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

// add a route to our configuration object
// also set availability flag based on cluster name
func (cfg *Config) AddRoute(clusterName string, path string, pathType string) {
	var availability uint8
	switch clusterName[len(clusterName)-2:] {
	case "in":
		availability = INTERNAL
	case "ex":
		availability = EXTERNAL
	case "ie":
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

// add an endpoint to our configuration object
func (cfg *Config) AddEndpoint(address string, clusterName string, port uint, region string, weight uint) {
	cfg.Endpoints[clusterName] = append(cfg.Endpoints[clusterName], &Endpoint{
		Address:     address,
		ClusterName: clusterName,
		Port:        port,
		Region:      region,
		Weight:      weight,
	})
}

func MergeConfigs(configs map[string]*Config) *Config {
	bigConfig := NewConfig()
	bigConfig.AddListener("127.0.0.1", "internal", 7777)
	bigConfig.AddListener("127.0.0.1", "external", 8888)

	for _, config := range configs {
		for _, l := range config.Listeners {
			for _, r := range l.Routes {
				bigConfig.Listeners[l.Name].Routes = append(bigConfig.Listeners[l.Name].Routes, r)
			}
		}
		for _, c := range config.Clusters {
			bigConfig.AddCluster(c.Name, c.Policy)
		}
		for _, r := range config.Routes {
			bigConfig.AddRoute(r.ClusterName, r.Path, r.Type)
		}
		for _, edps := range config.Endpoints {
			for _, e := range edps {
				bigConfig.AddEndpoint(e.Address, e.ClusterName, e.Port, e.Region, e.Weight)
			}
		}
	}

	return bigConfig
}
