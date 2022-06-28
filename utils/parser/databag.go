package parser

import (
	"fmt"
	"strconv"
	"strings"

	univcfg "github.com/fmgornick/dynamic-envoy/utils/config/universal"
	usercfg "github.com/fmgornick/dynamic-envoy/utils/config/user"
)

// parser has all the databags and an instance of our resource
// uses the databags to create the resource
type BagParser struct {
	Bags   []usercfg.Bag  // user configuration (input)
	Config univcfg.Config // universal configuration (output)
}

// assuming our parser contains a valid array of bags...
// we create 2 listener configurations and add all the routes, clusters, and endpoints
func Parse(bags []usercfg.Bag) (*univcfg.Config, error) {
	// initialize bag parser variables
	var bp BagParser
	bp.Config = *univcfg.NewConfig()
	bp.Bags = bags

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

// NOTE: always returns nil, might want to change up implementation a bit
// add listeners to listener map
func (bp *BagParser) AddListeners() error {
	// hard code listener creation
	// if given data bags, then it's assumed there will only be 2 listeners
	bp.Config.AddListener("127.0.0.1", "internal", 7777)
	bp.Config.AddListener("127.0.0.1", "external", 8888)
	return nil
}

// add clusters to cluster map
func (bp *BagParser) AddClusters() error {
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			// create cluster name from bag id / path
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				return err
			}

			// call universal configs add cluster method to append to our  cluster configs
			bp.Config.AddCluster(clusterName, "round_robin")
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
				return err
			}
			// check if specific path provided, otherwise get path from bag id
			if backend.Match.Path.Pattern == "" {
				bp.Config.AddRoute(clusterName, "/"+strings.Replace(bag.Id, "-", "/", -1), "exact")
			} else {
				bp.Config.AddRoute(clusterName, backend.Match.Path.Pattern, backend.Match.Path.Type)
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
			if bp.Config.Routes[name[:len(name)-3]+"-in"] == nil {
				bp.Config.Listeners["internal"].Routes = append(bp.Config.Listeners["internal"].Routes, name)
			}
			if bp.Config.Routes[name[:len(name)-3]+"-ex"] == nil {
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
				return err
			}
			// if server doesn't have any endpoints, then we don't want to delete the cluster
			if len(backend.Server.Endpoints) == 0 {
				delete(bp.Config.Clusters, clusterName)
			}
			for _, endpoint := range backend.Server.Endpoints {
				var port uint
				var address string
				// check if a port is specified in the url
				// if there is a route, then assign it to our port variable and remove it from the address string
				// otherwise, just leave the address as is and assign as 443
				split := strings.Split(endpoint.Address, ":")

				if len(split) == 3 {
					address = split[0] + ":" + split[1]
					p, err := strconv.Atoi(split[2])
					if err != nil {
						port = 443
					} else {
						port = uint(p)
					}
				} else if endpoint.Port == 0 {
					address = endpoint.Address
					port = 443
				} else {
					address = endpoint.Address
					port = endpoint.Port
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
	var aBag string
	var aBack string

	var name string
	// if a path is given, then we want to make it our new cluster id
	if backend.Match.Path.Pattern == "" {
		name = bag.Id
	} else {
		name = strings.Replace(backend.Match.Path.Pattern, "/", "-", -1)[1:]
	}

	// add extension for the availability of the bag
	switch len(bag.Availability) {
	case 0:
		aBag = "ie"
	case 1:
		if bag.Availability[0] == "internal" || bag.Availability[0] == "external" {
			aBag = bag.Availability[0][:2]
		} else {
			return "", fmt.Errorf("invalid element in bag availability array")
		}
	case 2:
		if (bag.Availability[0] == "internal" || bag.Availability[0] == "external") &&
			(bag.Availability[1] == "internal" || bag.Availability[1] == "external") {
			aBag = "ie"
		} else {
			return "", fmt.Errorf("invalid element in bag availability array")
		}
	default:
		return "", fmt.Errorf("invalid element in bag availability array")
	}

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
		newName = name + "-" + aBack
	case false:
		if aBag == "ie" {
			newName = name + "-" + aBack
		} else if aBack == "ie" {
			newName = name + "-" + aBag
		} else {
			return "", fmt.Errorf("bag and backend have conflicting availabilities")
		}
	}

	return newName, nil
}
