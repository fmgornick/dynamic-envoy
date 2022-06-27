package parser

// takes specific type of user configuration and returns universal configuration
type Parser interface {
	// takes the user configuration and outputs universal config
	Parse() error

	// take listeners from user config and convert them to our universal config
	AddListeners() error

	// take clusters from user config and convert them to our universal config
	AddClusters() error

	// take routes from user config and convert them to our universal config
	AddRoutes() error

	// take endpoints from user config and convert them to our universal config
	AddEndpoints() error
}
