package config

import (
	"io/ioutil"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseConfigTask struct {
	m *boot.Model
}

func ParseConfigInfo() *parseConfigTask {
	return &parseConfigTask{}
}

func (t *parseConfigTask) Run(c *boot.Config, m *boot.Model) (*TiDBConfigInfo, *TiKVConfigInfo, *PDConfigInfo) {
	t.m = m

	tidb := t.parseTiDBConfigInfo(path.Join(c.Src, "config", "tidb"))
	tikv := t.parseTiKVConfigInfo(path.Join(c.Src, "config", "tikv"))
	pd := t.parsePDConfigInfo(path.Join(c.Src, "config", "pd"))

	return tidb, tikv, pd
}

func (t *parseConfigTask) parseTiDBConfigInfo(dir string) *TiDBConfigInfo {
	info := TiDBConfigInfo{}

	insts, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Warn("parse tidb config:", err)
		return nil
	}

	for _, inst := range insts {
		if c := t.parseTiDBConfig(path.Join(dir, inst.Name(), "tidb.toml")); c != nil {
			info[inst.Name()] = c
		}
	}

	return &info
}

func (t *parseConfigTask) parseTiDBConfig(file string) *TiDBConfig {
	config := TiDBConfig{}

	if _, err := toml.DecodeFile(file, &config); err != nil {
		log.Error("decode tidb config:", err)
		return nil
	}

	return &config
}

func (t *parseConfigTask) parseTiKVConfigInfo(dir string) *TiKVConfigInfo {
	info := TiKVConfigInfo{}

	insts, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Warn("parse tikv config:", err)
		return nil
	}

	for _, inst := range insts {
		if c := t.parseTiKVConfig(path.Join(dir, inst.Name(), "tikv.toml")); c != nil {
			info[inst.Name()] = c
		}
	}

	return &info
}

func (t *parseConfigTask) parseTiKVConfig(file string) *TiKVConfig {
	config := TiKVConfig{}

	if _, err := toml.DecodeFile(file, &config); err != nil {
		log.Error("decode tikv config:", err)
		return nil
	}

	return &config
}

func (t *parseConfigTask) parsePDConfigInfo(dir string) *PDConfigInfo {
	info := PDConfigInfo{}

	insts, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Warn("parse pd config:", err)
		return nil
	}

	for _, inst := range insts {
		if c := t.parsePDConfig(path.Join(dir, inst.Name(), "pd.toml")); c != nil {
			info[inst.Name()] = c
		}
	}

	return &info
}

func (t *parseConfigTask) parsePDConfig(file string) *PDConfig {
	config := PDConfig{}

	if _, err := toml.DecodeFile(file, &config); err != nil {
		log.Error("decode pd config:", err)
		return nil
	}

	return &config
}
