package blockchain

type IProvider interface {
	Init(string, bool, bool) error
	Create(string, bool, bool, bool, string) error
	Join(string, string, string, bool, bool) error
	Leave(string, string,  bool, bool) error
	// WriteConfigs() error
	// GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
