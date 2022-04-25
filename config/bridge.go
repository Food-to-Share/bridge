package config

import (
	"bytes"
	"text/template"
)

type BridgeConfig struct {
	UsernameTemplate    string `yaml:"username_template"`
	DisplaynameTemplate string `yaml:"displayname_template"`

	CommandPrefix string `yaml:"command_prefix"`

	usernameTemplate    *template.Template `yaml:"-"`
	displaynameTemplate *template.Template `yaml:"-"`
}

type umBridgeConfig BridgeConfig

func (bc *BridgeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal((*umBridgeConfig)(bc))
	if err != nil {
		return err
	}

	bc.usernameTemplate, err = template.New("username").Parse(bc.UsernameTemplate)
	if err != nil {
		return err
	}

	bc.displaynameTemplate, err = template.New("displayname").Parse(bc.DisplaynameTemplate)
	return err
}

type DisplaynameTemplateArgs struct {
	Displayname string
}

type UsernameTemplateArgs struct {
	UserID string
}

func (bc BridgeConfig) FormatDisplayname(displayname string) string {
	var buf bytes.Buffer
	bc.displaynameTemplate.Execute(&buf, DisplaynameTemplateArgs{
		Displayname: displayname,
	})
	return buf.String()
}

func (bc BridgeConfig) FormatUsername(userID string) string {
	var buf bytes.Buffer
	bc.usernameTemplate.Execute(&buf, UsernameTemplateArgs{
		UserID: userID,
	})
	return buf.String()
}

func (bc BridgeConfig) MarshalYAML() (interface{}, error) {
	bc.DisplaynameTemplate = bc.FormatDisplayname("{{.Displayname}}")
	bc.UsernameTemplate = bc.FormatUsername("{{.UserID}}")
	return bc, nil
}
