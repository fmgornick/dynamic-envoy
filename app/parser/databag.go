package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	univcfg "github.com/fmgornick/dynamic-proxy/app/config/universal"
	usercfg "github.com/fmgornick/dynamic-proxy/app/config/user"
)

var schemes map[string]uint = map[string]uint{
	"ftp":    20,
	"gopher": 70,
	"http":   80,
	"https":  443,
	"imap":   143,
	"ldap":   389,
	"nfs":    2049,
	"nntp":   119,
	"pop":    110,
	"smtp":   25,
	"telnet": 23,
}

var policy map[string]string = map[string]string{
	"":          "round_robin",
	"static-rr": "round_robin",
	"leastconn": "least_request",
}

// parser has all the databags and an instance of our resource
// uses the databags to create the resource
type BagParser struct {
	Bags         []usercfg.Bag  // user configuration (input)
	Config       univcfg.Config // universal configuration (output)
	ListenerInfo univcfg.ListenerInfo
}

// assuming our parser contains a valid array of bags...
// we create 2 listener configurations and add all the routes, clusters, and endpoints
func Parse(bags []usercfg.Bag, l univcfg.ListenerInfo) (*univcfg.Config, error) {
	// initialize bag parser variables
	var bp BagParser
	bp.Bags = bags
	bp.Config = *univcfg.NewConfig()
	bp.ListenerInfo = l

	var err error
	err = bp.AddListeners()
	if err != nil {
		return nil, fmt.Errorf("unable to add listeners: %+v", err)
	}
	err = bp.AddClusters()
	if err != nil {
		return nil, fmt.Errorf("unable to add clusters: %+v", err)
	}
	err = bp.AddEndpoints()
	if err != nil {
		return nil, fmt.Errorf("unable to add endpoints: %+v", err)
	}
	err = bp.AddRoutes()
	if err != nil {
		return nil, fmt.Errorf("unable to add routes: %+v", err)
	}

	// return new universal config for further processing
	return &bp.Config, nil
}

// add listeners to listener map
func (bp *BagParser) AddListeners() error {
	// if given data bags, then it's assumed there will only be 2 listeners
	bp.Config.AddListener(bp.ListenerInfo.InternalAddress, "internal", bp.ListenerInfo.InternalPort)
	bp.Config.AddListener(bp.ListenerInfo.ExternalAddress, "external", bp.ListenerInfo.ExternalPort)
	return nil
}

// add clusters to cluster map
func (bp *BagParser) AddClusters() error {
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			// create cluster name from bag id / path
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				if err.Error() == "found gcp-external only api: "+bag.Id {
					break
				} else {
					return err
				}
			}
			// call universal configs add cluster method to append to our  cluster configs
			bp.Config.AddCluster(clusterName, policy[backend.Balance])
		}
	}
	return nil
}

// add routes to listener's route array
// add routes to route map
func (bp *BagParser) AddRoutes() error {
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				if err.Error() == "found gcp-external only api: "+bag.Id {
					break
				} else {
					return err
				}
			}
			// check if specific path provided, otherwise get path from bag id
			bagPath := "/" + strings.Replace(bag.Id, "-", "/", -1)
			if backend.Match.Path.Pattern == "" {
				bp.Config.AddRoute(clusterName, bagPath, "starts_with")
			} else {
				if backend.Match.Path.Type == "" {
					bp.Config.AddRoute(clusterName, backend.Match.Path.Pattern, "starts_with")
				} else {
					bp.Config.AddRoute(clusterName, backend.Match.Path.Pattern, backend.Match.Path.Type)
				}
			}
		}
	}
	// add the strictly internal or external routes to our listener route array
	for name, route := range bp.Config.Routes {
		if route.Availability == uint8(univcfg.INTERNAL) {
			bp.Config.Listeners["internal"].Routes = append(bp.Config.Listeners["internal"].Routes, name)
		} else if route.Availability == uint8(univcfg.EXTERNAL) {
			bp.Config.Listeners["external"].Routes = append(bp.Config.Listeners["external"].Routes, name)
		}
	}
	// now add the "available for both" routes if their more specific route isn't already in the array
	for name, route := range bp.Config.Routes {
		if route.Availability == uint8(univcfg.BOTH) {
			if bp.Config.Routes[name[:len(name)-2]+"in"] == nil {
				bp.Config.Listeners["internal"].Routes = append(bp.Config.Listeners["internal"].Routes, name)
			}
			if bp.Config.Routes[name[:len(name)-2]+"ex"] == nil {
				bp.Config.Listeners["external"].Routes = append(bp.Config.Listeners["external"].Routes, name)
			}
		}
	}
	return nil
}

// add endpoints to endpoint map
func (bp *BagParser) AddEndpoints() error {
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			// retrieve name of cluster the endpoint maps to
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				if err.Error() == "found gcp-external only api: "+bag.Id {
					break
				} else {
					return err
				}
			}
			// if server doesn't have any endpoints, then we don't want to delete the cluster
			if len(backend.Server.Endpoints) == 0 {
				delete(bp.Config.Clusters, clusterName)
			}
			for _, endpoint := range backend.Server.Endpoints {
				// check if a port is specified in the url
				// if there is a route, then assign it to our port variable and remove it from the address string
				// otherwise, just leave the address as is and assign as 80
				var address string
				var port uint

				var addr string
				if strings.Contains(endpoint.Address, "://") {
					addr = endpoint.Address
				} else {
					addr = "https://" + endpoint.Address
				}

				u, err := url.Parse(addr)
				if err != nil {
					return fmt.Errorf("error parsing url: %+v", err)
				}
				if _, ok := schemes[u.Scheme]; !ok {
					return fmt.Errorf("invalid schema: %s", u.Scheme)
				}

				address = u.Hostname() + u.Path
				if address[len(address)-1] == '/' {
					address = address[:len(address)-1]
				}
				portString := u.Port()
				if portString == "" {
					if endpoint.Port == 0 {
						port = uint(schemes[u.Scheme])
					} else {
						port = endpoint.Port
					}
				} else {
					p, _ := strconv.Atoi(portString)
					port = uint(p)
				}
				// add endpoints to endpoint map
				bp.Config.AddEndpoint(address, clusterName, port, endpoint.Region, endpoint.Weight)
			}
		}
	}
	return nil
}

// helper: rename cluster to provide information on which listeners have access
func getClusterName(bag usercfg.Bag, backend usercfg.Backend) (string, error) {
	var newName string

	var name string
	// if a path is given, then we want to make it our new cluster id
	if backend.Match.Path.Pattern == "" {
		name = bag.Id
	} else {
		name = strings.Replace(backend.Match.Path.Pattern, "/", "-", -1)[1:]
	}

	zone_mask := 0
	if len(bag.Availability) == 0 {
		zone_mask |= 3
	}

	for _, zone := range bag.Availability {
		switch zone {
		case "internal":
			zone_mask |= 1
		case "external":
			zone_mask |= 2
		case "gcp-external":
			// pass
		default:
			return "", fmt.Errorf("invalid availability: %s", zone)
		}
	}

	var aBag string
	switch zone_mask {
	case 0:
		// it is legal for APIs to be gcp-external only
		// we do not yet handle that case at higher calling functions
		return "", fmt.Errorf("found gcp-external only api: %s", bag.Id)
	case 1:
		aBag = "in"
	case 2:
		aBag = "ex"
	case 3:
		aBag = "ie"
	default:
		return "", fmt.Errorf("unexpected availability bitmask: %d", zone_mask)
	}

	var aBack string
	// add extension for the availability of the backend
	switch len(backend.Availability) {
	case 0:
		aBack = "ie"
	case 1:
		if backend.Availability[0] == "internal" || backend.Availability[0] == "external" {
			aBack = backend.Availability[0][:2]
		} else {
			return "", fmt.Errorf("invalid element in backend availability array")
		}
	case 2:
		if (backend.Availability[0] == "internal" || backend.Availability[0] == "external") &&
			(backend.Availability[1] == "internal" || backend.Availability[1] == "external") {
			aBack = "ie"
		} else {
			return "", fmt.Errorf("invalid element in backend availability array")
		}
	default:
		return "", fmt.Errorf("invalid element in backend availability array")
	}

	// compare the two extensions to create the actual extension for the cluster
	switch aBag == aBack {
	case true:
		if name == "" {
			newName = aBack
		} else {
			newName = name + "-" + aBack
		}
	case false:
		if aBag == "ie" {
			if name == "" {
				newName = aBack
			} else {
				newName = name + "-" + aBack
			}
		} else if aBack == "ie" {
			if name == "" {
				newName = aBag
			} else {
				newName = name + "-" + aBag
			}
		} else {
			return "", fmt.Errorf("bag and backend have conflicting availabilities")
		}
	}

	return newName, nil
}
