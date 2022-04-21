package config

import (
	"bytes"
	"text/template"
)

type BridgeConfig struct {
	RawUsernameTemplate    string             `yaml:"username_template"`
	RawDisplaynameTemplate string             `yaml:"displayname_template"`
	UsernameTemplate       *template.Template `yaml:"-"`
	DisplaynameTemplate    *template.Template `yaml:"-"`
}

func (bc BridgeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(bc)
	if err != nil {
		return err
	}

	bc.UsernameTemplate, err = template.New("username").Parse(bc.RawUsernameTemplate)
	if err != nil {
		return err
	}

	bc.DisplaynameTemplate, err = template.New("displayname").Parse(bc.RawDisplaynameTemplate)
	return err
}

type DisplaynameTemplateArgs struct {
	Displayname string
}

type UsernameTemplateArgs struct {
	Receiver string
	UserID   string
}

func (bc BridgeConfig) FormatDisplayname(displayname string) string {
	var buf bytes.Buffer
	bc.DisplaynameTemplate.Execute(&buf, DisplaynameTemplateArgs{
		Displayname: displayname,
	})
	return buf.String()
}

func (bc BridgeConfig) FormatUsername(receiver, userID string) string {
	var buf bytes.Buffer
	bc.UsernameTemplate.Execute(&buf, UsernameTemplateArgs{
		Receiver: receiver,
		UserID:   userID,
	})
	return buf.String()
}

func (bc BridgeConfig) MarshalYAML() (interface{}, error) {
	bc.RawDisplaynameTemplate = bc.FormatDisplayname("{{.Displayname}}")
	bc.RawUsernameTemplate = bc.FormatUsername("{{.Receiver}}", "{{.UserID}}")
	return bc, nil
}
