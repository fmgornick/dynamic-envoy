package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fmgornick/dynamic-envoy/config/utils/univcfg"
	"github.com/fmgornick/dynamic-envoy/config/utils/usercfg"
)

// parser has all the databags and an instance of our resource
// uses the databags to create the resource
type BagParser struct {
	Bags   []usercfg.Bag
	Config univcfg.Config
}

// assuming our parser contains a valid array of bags...
// we create 2 listener configurations and add all the routes, clusters, and endpoints
func Parse(bags []usercfg.Bag) (*univcfg.Config, error) {
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

	return &bp.Config, nil
}

// NOTE: always returns nil, might want to change up implementation a bit
func (bp *BagParser) AddListeners() error {
	bp.Config.AddListener("127.0.0.1", "internal", 7777)
	bp.Config.AddListener("127.0.0.1", "external", 8888)
	return nil
}

func (bp *BagParser) AddClusters() error {
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				return err
			}

			bp.Config.AddCluster(clusterName, "round_robin")
		}
	}
	return nil
}

func (bp *BagParser) AddEndpoints() error {
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				return err
			}
			for i, endpoint := range backend.Server.Endpoints {
				endpointName := clusterName + "-" + strconv.Itoa(i)
				var port uint
				var address string
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
				bp.Config.AddEndpoint(address, clusterName, endpointName, port, endpoint.Region, endpoint.Weight)
				bp.Config.Clusters[clusterName].Endpoints = append(bp.Config.Clusters[clusterName].Endpoints, endpointName)
			}
		}
	}
	return nil
}

// add routes to listener's route array
// add routes to route map
func (bp *BagParser) AddRoutes() error {
	// iterate through all the backends and create a route
	for _, bag := range bp.Bags {
		for _, backend := range bag.Backends {
			clusterName, err := getClusterName(bag, backend)
			if err != nil {
				return err
			}
			if backend.Match.Path.Pattern == "" {
				bp.Config.AddRoute(clusterName, "/"+strings.Replace(bag.Id, "-", "/", -1), "exact")
			} else {
				bp.Config.AddRoute(clusterName, backend.Match.Path.Pattern, backend.Match.Path.Type)
			}
		}
	}
	for routeName := range bp.Config.Routes {
		if routeName[len(routeName)-3:] == "-in" {
			bp.Config.Listeners["internal"].Routes = append(bp.Config.Listeners["internal"].Routes, routeName)
		} else if routeName[len(routeName)-3:] == "-ex" {
			bp.Config.Listeners["external"].Routes = append(bp.Config.Listeners["external"].Routes, routeName)
		}
	}
	for routeName := range bp.Config.Routes {
		if routeName[len(routeName)-3:] == "-ie" {
			if bp.Config.Routes[routeName[:len(routeName)-3]+"-in"] == nil {
				bp.Config.Listeners["internal"].Routes = append(bp.Config.Listeners["internal"].Routes, routeName)
			}
			if bp.Config.Routes[routeName[:len(routeName)-3]+"-ex"] == nil {
				bp.Config.Listeners["external"].Routes = append(bp.Config.Listeners["external"].Routes, routeName)
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
	if backend.Match.Path.Pattern == "" {
		name = bag.Id
	} else {
		name = strings.Replace(backend.Match.Path.Pattern, "/", "-", -1)[1:]
	}

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
