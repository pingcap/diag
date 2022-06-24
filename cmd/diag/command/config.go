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

package command

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/spf13/cobra"
)

type ClinicConfig struct {
	//	Endpoint string `toml:"endpoint,omitempty"`
	//	Cert     string `toml:"cert,omitempty"`
	Region string `toml:"region,omitempty"`
	Token  string `toml:"token,omitempty"`
}

type DiagConfig struct {
	Clinic ClinicConfig `toml:"clinic,omitempty"`
}

const RegionInfo = `Clinic Server provides the following two regions to store your diagnostic data:
[CN] region: Data stored in China Mainland, domain name : https://clinic.pingcap.com.cn
[US] region: Data stored in USA ,domain name : https://clinic.pingcap.com`

func newConfigCmd() *cobra.Command {
	var unset, show bool
	cmd := &cobra.Command{
		Use:   "config <key> [value] [--unset]",
		Short: "set an individual value in diag configuration file",
		Long: `set an individual value in diag configuration file, like
  "diag config clinic.token xxxxxxxxxx"
if not specify key nor value, an interactive interface will be used to set necessary configuration`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if show {
				confStr, err := os.ReadFile(spec.ProfilePath("diag.toml"))
				if err != nil {
					return err
				}
				fmt.Println(string(confStr))
			} else if unset {
				if len(args) != 1 {
					return cmd.Help()
				}
				err := diagConfig.Unset(args[0])
				if err != nil {
					return err
				}
			} else {
				switch len(args) {
				case 0:
					diagConfig.interactiveSet()
				case 2:
					err := diagConfig.Set(args[0], args[1])
					if err != nil {
						return err
					}
				default:
					return cmd.Help()
				}
			}
			return diagConfig.Save()
		},
	}

	cmd.PersistentFlags().BoolVar(&unset, "unset", false, "unset diag configuration")
	cmd.PersistentFlags().BoolVar(&show, "show", false, "show diag configuration")

	return cmd
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
	reflectV := reflect.ValueOf(&diagConfig).Elem()
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
	value, err := validateConfigKey(key, value)
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
	reflectV := reflect.ValueOf(&diagConfig).Elem()
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

func (c *DiagConfig) interactiveSet() {
	fmt.Println("diag upload need token which you could get from https://clinic.pingcap.com.cn")
	fmt.Print("please input your token:")
	fmt.Scanf("%s", &c.Clinic.Token)
}

func validateConfigKey(key, value string) (string, error) {
	switch key {
	case "clinic.region":
		if value != "cn" && value != "us" {
			return value, fmt.Errorf("%s cannot be %s, it can only be CN or US\n%s", key, value, RegionInfo)
		}
	}
	return strings.ToUpper(value), nil
}
