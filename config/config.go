package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/id"
)

var ExampleConfig string

type Config struct {
	Homeserver struct {
		Address                       string `yaml:"address"`
		Domain                        string `yaml:"domain"`
		Asmux                         bool   `yaml:"asmux"`
		StatusEndpoint                string `yaml:"status_endpoint"`
		MessageSendCheckpointEndpoint string `yaml:"message_send_checkpoint_endpoint"`
		AsyncMedia                    bool   `yaml:"async_media"`
	} `yaml:"homeserver"`

	AppService struct {
		Address  string `yaml:"address"`
		Hostname string `yaml:"hostname"`
		Port     uint16 `yaml:"port"`

		Database DatabaseConfig `yaml:"database"`

		Provisioning struct {
			Prefix       string `yaml:"prefix"`
			SharedSecret string `yaml:"shared_secret"`
			SegmentKey   string `yaml:"segment_key"`
		} `yaml:"provisioning"`

		ID  string `yaml:"id"`
		Bot struct {
			Username    string `yaml:"username"`
			Displayname string `yaml:"displayname"`
			Avatar      string `yaml:"avatar"`

			ParsedAvatar id.ContentURI `yaml:"-"`
		} `yaml:"bot"`

		EphemeralEvents bool `yaml:"ephemeral_events"`

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

func (config *Config) MakeAppService() (*appservice.AppService, error) {
	as := appservice.Create()
	as.HomeserverDomain = config.Homeserver.Domain
	as.HomeserverURL = config.Homeserver.Address
	as.Host.Hostname = config.AppService.Hostname
	as.Host.Port = config.AppService.Port
	as.MessageSendCheckpointEndpoint = config.Homeserver.MessageSendCheckpointEndpoint
	as.DefaultHTTPRetries = 4
	var err error
	as.Registration, err = config.GetRegistration()
	return as, err
}

type DatabaseConfig struct {
	Type string `yaml:"type"`
	URI  string `yaml:"uri"`

	MaxOpenConns int `yaml:"max_open_conns"`
	MaxIdleConns int `yaml:"max_idle_conns"`

	ConnMaxIdleTime string `yaml:"conn_max_idle_time"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}
