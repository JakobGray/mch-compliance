package cmd

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Exemptions map[string][]Exemption `json:"exemptions"`
}

type Exemption struct {
	Name   string   `json:"name"`
	Checks []string `json:"checks"`
}

func readConfig(file string) (*Config, error) {
	b, err := ioutil.ReadFile(file)

	c := &Config{}
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return c, err
	}

	return c, nil
}
