package configs

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

var (
	config     *Config
	configLock = &sync.RWMutex{}
)

type Config struct {
	Server   SeverConfig   `mapstructure:"server"`
	Chain    ChainConfig   `mapstructure:"chain"`
	Scanner  ScannerConfig `mapstructure:"scanner"`
	RabbitMQ MQConfig      `mapstructure:"rabbitmq"`
}

type SeverConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

type ChainConfig struct {
	ChainID   uint64   `mapstructure:"chain_id"`
	Provider  string   `mapstructure:"provider"`
	Contracts []string `mapstructure:"contracts"`
}

type ScannerConfig struct {
	StartBlock        uint64 `mapstructure:"start_block"`
	ConcurrentWorkers int    `mapstructure:"concurrent_workers"`
	MarkFile          string `mapstructure:"mark_file"`
}

type MQConfig struct {
	URL       string `mapstructure:"url"`
	QueueName string `mapstructure:"queue_name"`
}

func LoadConfig() error {
	if err := gotenv.Load(); err != nil {
		return fmt.Errorf("error loading .env file: %w", err)
	}

	// 获取当前环境
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev" // 默认为开发环境
	}

	// 设置配置文件路径
	configPath := fmt.Sprintf("./configs/config.%s.yaml", env)

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := updateConfig(); err != nil {
		return err
	}

	// 设置配置文件热重载
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		if err := updateConfig(); err != nil {
			fmt.Printf("Error reloading config: %v\n", err)
		} else {
			fmt.Println("Config successfully reloaded")
		}
	})

	return nil
}

func updateConfig() error {
	var newConfig Config
	if err := viper.Unmarshal(&newConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	configLock.Lock()
	config = &newConfig
	configLock.Unlock()

	if err := checkConfig(); err != nil {
		return err
	}

	return nil
}

func checkConfig() error {

	if config.Chain.ChainID == 0 {
		return errors.New("chain_id is not configure")
	}
	if config.Chain.Provider == "" {
		return errors.New("provider is not configure")
	}

	if len(config.Chain.Contracts) == 0 {
		return errors.New("contracts is not configure")
	}

	return nil

}

func GetConfig() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

func init() {

	err := LoadConfig()
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

}
