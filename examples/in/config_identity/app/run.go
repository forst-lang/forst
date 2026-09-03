package app

import "example.com/config_identity/config"

func Run(cfg config.Config) error {
	_ = cfg.Port
	return nil
}
