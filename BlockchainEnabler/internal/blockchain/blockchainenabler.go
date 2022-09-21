package blockchain

type IProvider interface {
	Init(string, bool, bool) error
	Create(string, bool, bool, string) error
	Join(string, string, string, bool, bool) error
	Leave(string, string, string, bool, bool) error
	Invite(string, bool, string) error
	Request(string, string, string, bool, string) error

	Accept(string, bool, string, bool) error
	Delete(string) error

	// WriteConfigs() error
	// GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
