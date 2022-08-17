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

package collector

import (
	"crypto/tls"
	"strings"

	"github.com/pingcap/diag/pkg/models"
	"go.etcd.io/etcd/client/pkg/v3/transport"
)

// buildTopoForManualCluster creates an abstract topo from user input
func buildTopoForManualCluster(cOpt *CollectOptions) (*models.TiDBCluster, error) {
	// build the abstract topology
	cls := &models.TiDBCluster{
		Version:    "unknown",
		Attributes: map[string]interface{}{},
	}

	cls.Attributes[AttrKeyPDEndpoint] = strings.Split(cOpt.ExtendedAttrs[AttrKeyPDEndpoint], ",")
	cls.Attributes[AttrKeyTiDBHost] = cOpt.ExtendedAttrs[AttrKeyTiDBHost]
	cls.Attributes[AttrKeyTiDBPort] = cOpt.ExtendedAttrs[AttrKeyTiDBPort]
	cls.Attributes[AttrKeyTiDBStatus] = cOpt.ExtendedAttrs[AttrKeyTiDBStatus]
	cls.Attributes[AttrKeyPromEndpoint] = strings.Split(cOpt.ExtendedAttrs[AttrKeyPromEndpoint], ",")

	return cls, nil
}

// tlsConfig generates a tls.Config from certificate files
func tlsConfig(ca, cert, key string) (*tls.Config, error) {
	// handle non-TLS clusters
	if cert == "" || key == "" {
		return nil, nil
	}

	return transport.TLSInfo{
		TrustedCAFile: ca,
		CertFile:      cert,
		KeyFile:       key,
	}.ClientConfig()
}
