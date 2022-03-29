// Copyright 2021 PingCAP, Inc.
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
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

type ClinicConfig struct {
//	Endpoint string `toml:"endpoint,omitempty"`
//	Cert     string `toml:"cert,omitempty"`
	Token    string `toml:"token,omitempty"`
}

type DiagConfig struct {
	Clinic ClinicConfig `toml:"clinic,omitempty"`
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [<options>]",
		Short: "config diag",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			switch len(args) {
			case 0:
				diagConfig.interactiveSet()
			case 2:
				err:= diagConfig.Set(args[0], args[1])
				if err != nil {
					return err
				}
			default:
				return cmd.Help()
			}
			return diagConfig.Save()
		},
	}

	return cmd
}

func (c *DiagConfig) Load() error {
	confPath := os.Getenv("TIUP_COMPONENT_DATA_DIR")
	if confPath == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir == "" {
			return fmt.Errorf("cannot get configure path")
		}
		confPath = filepath.Join(homeDir, ".tiup", "storage", "diag")
	}
	confPath = filepath.Join(confPath, "diag.toml")
	_, err := toml.DecodeFile(confPath, c)
	return err
	
}

func (c *DiagConfig) Save() error {
	confPath := os.Getenv("TIUP_COMPONENT_DATA_DIR")
	if confPath == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir == "" {
			return fmt.Errorf("cannot get configure path")
		}
		confPath = filepath.Join(homeDir, ".tiup", "storage", "diag")
	}
	confPath = filepath.Join(confPath, "diag.toml")
	f, err := os.OpenFile(confPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	enc := toml.NewEncoder(f)
	return enc.Encode(c)
	
}

func (c *DiagConfig) Set(key, value string) error{
	reflectV := reflect.ValueOf(&diagConfig).Elem()
	keys := strings.Split(key, ".")
	for _, k := range keys {
		if reflectV.Kind() != reflect.Struct {
			return fmt.Errorf("%s is not a valid diag configure key", key)
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
			return fmt.Errorf("%s is not a valid diag configure key", key)
		}
	}
	if reflectV.CanSet() && reflectV.Kind() != reflect.Struct{
		reflectV.SetString(value)
	} else {
		return fmt.Errorf("%s is not a valid diag configure key", key)
	}
	return nil
}

func (c *DiagConfig) interactiveSet() {
	fmt.Println("diag upload need token which you could get from https://clinic.pingcap.com")
	fmt.Print("please input your token:")
	fmt.Scanf("%s", &c.Clinic.Token)
}