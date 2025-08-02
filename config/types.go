package config

type AuthConfig struct {
	SecretKey string `yaml:"secret_key"`
	Username  string `yaml:"username"`
	Timeout   int    `yaml:"timeout"`
}

type Server struct {
	UseDirectCamera bool   `yaml:"use_direct_camera"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
}
