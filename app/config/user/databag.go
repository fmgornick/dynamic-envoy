package usercfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Bag struct {
	Availability []string  `json:"availability"` // "internal", "external", or both
	Backends     []Backend `json:"backends"`     // "match" maps to route, "availability" maps to listener, the rest go to cluster
	Groups       []string  `json:"groups"`       // not my problem for now
	Id           string    `json:"id"`           // url path swapped with dashes
}

type Backend struct {
	Availability  []string    `json:"availability"`         // "internal", "external", or both.  DEFAULT TO BOTH
	Balance       string      `json:"balance"`              // load balancing policy, default should be round robin
	HealthCheck   HealthCheck `json:"healthcheck"`          // don't worry about this for now
	IgnoreDefault bool        `json:"ignore_default_match"` // set to true if ignoring default match pattern
	Match         Match       `json:"match"`                // if match set, then listener should check route paths until finding a match
	RateLimit     RateLimit   `json:"rate_limit"`           // don't worry about this one either
	Server        Server      `json:"servers"`              // basically a cluster
}

type Server struct {
	Endpoints []Endpoint // server is essentially a cluster with 1+ endpoints
}

type Endpoint struct {
	Address string `json:"address"` // where the user actually gets sent
	Port    uint   `json:"port"`    // default to 443
	Region  string `json:"region"`  // "global", "ttc", or "ttce"
	Weight  uint   `json:"weight"`  // should default to 0 unless "Balance" set to weighted round robin
}

// i don't really know what the healthcheck does (for now)
type HealthCheck struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Type   string `json:"type"`
}

type Match struct {
	Path Path `json:"path"` // info on how to match the url
}

type Path struct {
	Pattern string `json:"pattern"` // url path, also cluster name
	Type    string `json:"type"`    // either "exact" or "starts_with"
}

type RateLimit struct {
	Count uint   `json:"count"` // number of times link accessed per second
	Field string `json:"field"` // don't needa worry bout rate limit right now
}

// turn json file into a resource object
func ParseFile(filename string) ([]Bag, error) {
	var bags []Bag
	var bag Bag

	// get directory contents
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ERROR - couldn't read file: %s\n", err)
	}

	// parse each file into a data bag
	err = json.Unmarshal(file, &bag)
	if err != nil {
		return nil, err
	}
	bags = append(bags, bag)

	return bags, nil
}
