package blockchain

type IProvider interface {
	Init(string, bool, bool, bool, bool) error
	Create(string, bool, bool, bool) error
	Join(string, bool, string, bool, bool) error
	Leave(string, string, string, bool, bool) error
	Sign(string, bool, string, bool, bool) error
	Add(string, bool, string,bool) error
	Delete(string,bool) error

	// WriteConfigs() error
	// GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
