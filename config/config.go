package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"maunium.net/go/mautrix/appservice"
)

type Config struct {
	Homeserver struct {
		Address string `yaml:"address"`
		Domain  string `yaml:"domain"`
	} `yaml:"homeserver"`

	AppService struct {
		Address  string `yaml:"address"`
		Hostname string `yaml:"hostname"`
		Port     uint16 `yaml:"port"`

		Database struct {
			Organization string `yaml:type`
			URI string `yaml: "uri`
		}

		ID  string `yaml:"id"`
		Bot struct {
			Username    string `yaml:"username"`
			Displayname string `yaml:"displayname"`
		} `yaml:"bot"`

		ASToken string `yaml:"as_token"`
		HSToken string `yaml:"hs_token"`
	} `yaml:"appservice"`

	Bridge BridgeConfig `yaml:"bridge"`

	Logging appservice.LogConfig `yaml:"logging"`
}

func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config = &Config{}
	err = yaml.Unmarshal(data, config)

	return config, err
}

func (config *Config) Save(path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0600)
}

func (config *Config) MakeAppservice() (*appservice.AppService, error) {
	as := appservice.Create()
	as.LogConfig = config.Logging
	as.HomeserverDomain = config.Homeserver.Domain
	as.HomeserverURL = config.Homeserver.Address
	var err error
	as.Registration, err = config.GetRegistration()
	return as, err
}
