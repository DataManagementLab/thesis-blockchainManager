package blockchain

type IProvider interface {
	Init(string,bool,bool) error
	Create(string,bool,bool) error
	Join(string, string, string, string, string,bool,bool) error
	Leave(string,string,string)error
	// WriteConfigs() error
	// GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
