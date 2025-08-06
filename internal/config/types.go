package config

type Auth struct {
	JwtSecretKey   string `yaml:"jwt_secret_key"`
	ExpirationTime int    `yaml:"expiration_time"`
	Users          []User `yaml:"users"`
}

type Server struct {
	UseDirectCamera bool   `yaml:"use_direct_camera"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
}

type User struct {
	Username       string `yaml:"username"`
	HashedPassword string `yaml:"hashed_password"`
}
