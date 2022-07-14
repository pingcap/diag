// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pingcap/tiup/pkg/cluster/spec"
)

// user specified diag config
type DiagConfig struct {
	Clinic ClinicConfig `toml:"clinic,omitempty"`
}

type ClinicConfig struct {
	//	Endpoint string `toml:"endpoint,omitempty"`
	//	Cert     string `toml:"cert,omitempty"`
	Region Region `toml:"region,omitempty"`
	Token  string `toml:"token,omitempty"`
}

func (c *DiagConfig) Load() error {
	confPath := spec.ProfilePath("diag.toml")
	_, err := toml.DecodeFile(confPath, c)
	return err
}

func (c *DiagConfig) Save() error {
	confPath := spec.ProfilePath("diag.toml")
	f, err := os.OpenFile(confPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	enc := toml.NewEncoder(f)
	return enc.Encode(c)
}

// only support string value now
func (c *DiagConfig) Set(key, value string) error {
	reflectV := reflect.ValueOf(c).Elem()
	keys := strings.Split(key, ".")
	for _, k := range keys {
		if reflectV.Kind() != reflect.Struct {
			return fmt.Errorf("%s is not a valid diag configuration key1", key)
		}
		num := reflectV.NumField()
		var i int
		for i = 0; i < num; i++ {
			if k == strings.Split(reflectV.Type().Field(i).Tag.Get("toml"), ",")[0] {
				reflectV = reflectV.Field(i)
				break
			}
		}
		if i == num {
			return fmt.Errorf("%s is not a valid diag configuration key11", key)
		}
	}
	value, err := ValidateConfigKey(key, value)
	if err != nil {
		return err
	}

	if reflectV.CanSet() && reflectV.Kind() != reflect.Struct {
		reflectV.SetString(value)
	} else {
		return fmt.Errorf("%s is not a valid diag configuration", key)
	}
	return nil
}

func (c *DiagConfig) Unset(key string) error {
	reflectV := reflect.ValueOf(c).Elem()
	keys := strings.Split(key, ".")
	for _, k := range keys {
		if reflectV.Kind() != reflect.Struct {
			return fmt.Errorf("%s is not a valid diag configuration key", key)
		}
		num := reflectV.NumField()
		var i int
		for i = 0; i < num; i++ {
			if k == strings.Split(reflectV.Type().Field(i).Tag.Get("toml"), ",")[0] {
				reflectV = reflectV.Field(i)
				break
			}
		}
		if i == num {
			return fmt.Errorf("%s is not a valid diag configuration key", key)
		}
	}

	reflectV.SetString("")
	return nil
}

func ValidateConfigKey(key, value string) (string, error) {
	switch key {
	case "clinic.region":
		if _, ok := Info.ClinicServers[Region(strings.ToUpper(value))]; !ok {
			return value, fmt.Errorf("%s cannot be %s, available region are [%s]", key, value, strings.Join(AvailableRegion, ","))
		}
		return strings.ToUpper(value), nil
	}
	return value, nil
}

func (c *DiagConfig) InteractiveSet() error {
	err := c.InteractiveSetRegion()
	if err != nil {
		return err
	}
	c.InteractiveSetToken()
	return nil
}

func (c *DiagConfig) InteractiveSetToken() {
	fmt.Printf("diag upload need token which you could get from %s\n", c.Clinic.Region.Endpoint())
	fmt.Print("please input your token:")
	fmt.Scanf("%s", &c.Clinic.Token)
}

func (c *DiagConfig) InteractiveSetRegion() error {
	var r string
	fmt.Println(infoText)
	fmt.Print("please choose region:")
	fmt.Scanf("%s", &r)
	r, err := ValidateConfigKey("clinic.region", r)
	if err != nil {
		return err
	}
	c.Clinic.Region = Region(r)
	return nil
}
