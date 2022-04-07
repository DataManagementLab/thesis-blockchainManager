package blockchain

type IProvider interface {
	Init(string) error
	Create(string,bool) error
	Join(string, string, string, string, string) error
	Leave(string,string,string)error
	// WriteConfigs() error
	// GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
