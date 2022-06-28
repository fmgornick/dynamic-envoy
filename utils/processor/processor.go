package processor

// takes generic user configuration and returns configuration for the proxy being used
type Processor interface {
	// take in the universal config and output something to be used by the specific proxy
	Process() error

	// take universal listener config and turn it to specify proxy listener
	MakeListeners() error

	// take universal cluster config and turn it to specify proxy listener
	MakeClusters() error

	// take universal route config and turn it to specify proxy listener
	MakeRoutes() error

	// take universal endpoint config and turn it to specify proxy listener
	MakeEndpoints() error
}
