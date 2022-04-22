package core

import (
	"github.com/creasty/defaults"
)

func (sp *ServicePort) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(sp); err != nil {
		return err
	}
	type alias ServicePort
	if err := unmarshal((*alias)(sp)); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(cfg); err != nil {
		return err
	}
	type alias Config
	if err := unmarshal((*alias)(cfg)); err != nil {
		return err
	}
	return nil
}

func (cwn *ClusterWithName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(cwn); err != nil {
		return err
	}
	type alias ClusterWithName
	if err := unmarshal((*alias)(cwn)); err != nil {
		return err
	}
	return nil
}
