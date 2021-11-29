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

package main

import (
	"github.com/pingcap/diag/k8s/server"
	"github.com/pingcap/diag/version"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	diagCmd *cobra.Command
	srvOpt  *server.Options
)

func init() {
	klog.InitFlags(nil)
	defer klog.Flush()

	srvOpt = &server.Options{}

	diagCmd = &cobra.Command{
		Use:     "diag",
		Short:   "The TiDB diagnostic collector and checker for Kubernetes.",
		Version: version.String(),
		RunE:    runServer,
	}

	diagCmd.Flags().StringVar(&srvOpt.Host, "host", "0.0.0.0", "listen address")
	diagCmd.Flags().IntVar(&srvOpt.Port, "port", 4917, "listen port")
	diagCmd.Flags().BoolVar(&srvOpt.Verbose, "verbose", false, "debug log")
}

func runServer(_ *cobra.Command, args []string) error {
	klog.Infof("started diag pod %s", version.String())

	srv, err := server.NewServer(srvOpt)
	if err != nil {
		return err
	}

	return srv.Run()
}

func main() {
	if err := diagCmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}
