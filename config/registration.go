package config

import (
	"regexp"

	"fmt"

	"maunium.net/go/mautrix/appservice"
)

func (config *Config) NewRegistration() (*appservice.Registration, error) {
	registration := appservice.CreateRegistration()

	err := config.copyToRegistration(registration)
	if err != nil {
		return nil, err
	}

	config.AppService.ASToken = registration.AppToken
	config.AppService.HSToken = registration.ServerToken

	return registration, nil
}

func (config *Config) GetRegistration() (*appservice.Registration, error) {
	registration := appservice.CreateRegistration()

	err := config.copyToRegistration(registration)
	if err != nil {
		return nil, err
	}

	registration.AppToken = config.AppService.ASToken
	registration.ServerToken = config.AppService.HSToken
	return registration, nil
}

func (config *Config) copyToRegistration(registration *appservice.Registration) error {
	registration.ID = config.AppService.ID
	registration.URL = config.AppService.Address
	falseVal := false
	registration.RateLimited = &falseVal
	registration.SenderLocalpart = config.AppService.Bot.Username

	userIDRegex, err := regexp.Compile(fmt.Sprintf("@%s:%s",
		config.Bridge.FormatUsername("[0-9]+"),
		config.Homeserver.Domain))
	if err != nil {
		return err
	}
	registration.Namespaces.RegisterUserIDs(userIDRegex, true)
	return nil
}
