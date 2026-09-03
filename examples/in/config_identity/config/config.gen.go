package config

// Emitted form of config.ft — named Config must match Forst typedef for FFI.
type Config struct {
	Port string
}

func Load() Config {
	return Config{Port: "8080"}
}
