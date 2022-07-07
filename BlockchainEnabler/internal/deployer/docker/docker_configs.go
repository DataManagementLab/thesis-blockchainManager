package docker

type HealthCheck struct {
	Test     []string `yaml:"test,omitempty"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
}

type LoggingConfig struct {
	Driver  string            `yaml:"driver,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
}

type ServiceDefinition struct {
	ServiceName string
	Service     *Service
	VolumeNames []string
}
type DockerNetworkName struct {
	DockerExternalNetworkName string `yaml:"name,omitempty"`
}

type DockerNetwork struct {
	DockerExternalNetwork *DockerNetworkName `yaml:"external,omitempty"`
}

type Service struct {
	ContainerName      string                       `yaml:"container_name,omitempty"`
	Image              string                       `yaml:"image,omitempty"`
	Build              string                       `yaml:"build,omitempty"`
	Command            string                       `yaml:"command,omitempty"`
	Environment        map[string]string            `yaml:"environment,omitempty"`
	Volumes            []string                     `yaml:"volumes,omitempty"`
	Ports              []string                     `yaml:"ports,omitempty"`
	DependsOn          map[string]map[string]string `yaml:"depends_on,omitempty"`
	HealthCheck        *HealthCheck                 `yaml:"healthcheck,omitempty"`
	Logging            *LoggingConfig               `yaml:"logging,omitempty"`
	WorkingDir         string                       `yaml:"working_dir,omitempty"`
	EntryPoint         []string                     `yaml:"entrypoint,omitempty"`
	EnvFile            string                       `yaml:"env_file,omitempty"`
	Expose             []int                        `yaml:"expose,omitempty"`
	DockerNetworkNames []string                     `yaml:"networks,omitempty"`
}

type DockerComposeConfig struct {
	Version  string                    `yaml:"version,omitempty"`
	Services map[string]*Service       `yaml:"services,omitempty"`
	Volumes  map[string]struct{}       `yaml:"volumes,omitempty"`
	Networks map[string]*DockerNetwork `yaml:"networks,omitempty"`
}

func CreateDockerCompose() *DockerComposeConfig {
	compose := &DockerComposeConfig{
		Version:  "2.1",
		Services: make(map[string]*Service),
		Volumes:  make(map[string]struct{}),
		Networks: make(map[string]*DockerNetwork),
	}
	return compose
}
