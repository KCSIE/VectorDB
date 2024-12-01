package settings

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf = new(AppConfig)

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Mode    string `mapstructure:"mode"`
	Version string `mapstructure:"version"`

	*LogConfig `mapstructure:"log"`
	*DBConfig  `mapstructure:"db"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

type DBConfig struct {
	PersistPath string `mapstructure:"persist_path"`
}

func Init() (err error) {
	// set config file path
	viper.SetConfigFile("./config.yaml")
	// read config info
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("viper failed to read config file, err:%v\n", err)
		return
	}

	// unmarshal config info to Config
	if err := viper.Unmarshal(Conf); err != nil {
		fmt.Printf("viper deserialization failed, err:%v\n", err)
	}

	// listen change of config file
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("configuration file has been changed!")
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("viper deserialization failed, err:%v\n", err)
		}
	})

	return
}
