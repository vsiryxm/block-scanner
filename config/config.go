package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Environment string `yaml:"environment"`
	Ethereum    struct {
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

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config struct {
		Environment string            `yaml:"environment"`
		Configs     map[string]Config `yaml:",inline"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// byteData, _ := json.MarshalIndent(config, "", "\t") //加t 格式化显示
	// fmt.Println("config===", string(byteData))

	env := config.Environment
	if env == "" {
		env = "dev"
	}

	cfg, ok := config.Configs[env]
	if !ok {
		return nil, fmt.Errorf("environment '%s' not found in config", env)
	}

	return &cfg, nil
}
