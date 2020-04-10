package main

import (
	"gopkg.in/ini.v1"
	"strconv"
)

type Config struct {
	RPCURL  string
	ChainId int
	Port    int

	Account string
	Priv    string

	LastBlock    uint64
	RegistryAddr string
}

var cfg *ini.File

func LoadConfiguration(filepath string) (*Config, error) {
	var err error
	cfg, err = ini.Load(filepath)
	if err != nil {
		return nil, err
	}

	config := new(Config)

	config.RPCURL = cfg.Section("network").Key("rpc_host").String()
	config.Port = cfg.Section("network").Key("port").MustInt(8081)
	config.ChainId = cfg.Section("network").Key("chain_id").MustInt(3)

	config.Account = cfg.Section("account").Key("name").String()
	config.Priv = cfg.Section("account").Key("priv").String()

	config.LastBlock = uint64(cfg.Section("extapi").Key("lastBlock").MustInt(0))
	config.RegistryAddr = cfg.Section("extapi").Key("registry").String()

	return config, nil
}

func SaveConfiguration(config *Config, filepath string) {
	cfg.Section("extapi").Key("lastBlock").SetValue(strconv.FormatUint(config.LastBlock-1, 10))
	cfg.SaveTo(filepath)
}
