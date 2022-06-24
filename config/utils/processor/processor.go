package processor

type Processor interface {
	// initialize processor and get data from universal configuration
	NewProcessor()

	// take universal listener config and turn it to specify proxy listener
	ConfigureListeners() error

	// take universal cluster config and turn it to specify proxy listener
	ConfigureClusters() error

	// take universal route config and turn it to specify proxy listener
	ConfigureRoutes() error

	// take universal endpoint config and turn it to specify proxy listener
	ConfigureEndpoints() error
}
