package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var config *Config

type Config struct {
	Environment string `yaml:"environment"`
	Ethereum    struct {
		ChainId           uint64   `yaml:"chain_id"`
		RPCURL            string   `yaml:"rpc_url"`
		ContractAddresses []string `yaml:"contract_addresses"`
	} `yaml:"ethereum"`
	RabbitMQ struct {
		URL       string `yaml:"url"`
		QueueName string `yaml:"queue_name"`
	} `yaml:"rabbitmq"`
	Scanner struct {
		StartBlock        uint64 `yaml:"start_block"`
		ConcurrentWorkers int    `yaml:"concurrent_workers"`
		MarkFile          string `yaml:"mark_file"`
	} `yaml:"scanner"`
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
}

func LoadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var cfg struct {
		Environment string            `yaml:"environment"`
		Configs     map[string]Config `yaml:",inline"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	env := cfg.Environment
	if env == "" {
		env = "dev"
	}

	tmp, ok := cfg.Configs[env]
	if !ok {
		return fmt.Errorf("environment '%s' not found in config", env)
	}

	config = &tmp
	return nil
}

func GetConfig() *Config {
	return config
}
