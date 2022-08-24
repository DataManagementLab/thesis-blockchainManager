package blockchain

type IProvider interface {
	Init(string, bool, bool) error
	Create(string, bool, bool, bool, string) error
	Join(string, string, string, bool, bool) error
	Leave(string, string, string, bool, bool) error
	Invite(string, string, string, bool, string) error
	Request(string, string, string, bool, string) error

	Accept(string, string, string, bool,string) error

	// WriteConfigs() error
	// GetDockerServiceDefinitions() []*docker.ServiceDefinition

	// Create()
	// Join()
	// Sign()
	// Upload()
	// UploadSmartContract()
}
