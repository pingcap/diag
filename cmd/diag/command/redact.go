// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pingcap/diag/pkg/redact"
	"github.com/spf13/cobra"
)

type redactOptions struct {
	addresses           []string
	output              string
	allIPs              bool
	logIPv4QuestionMark bool
	inPlace             bool
	mapOutput           bool
	keepIPv4Octets      int
	workers             int
}

type mapExportValue bool

func (v *mapExportValue) Set(value string) error {
	switch strings.ToUpper(value) {
	case "Y":
		*v = true
	case "N":
		*v = false
	default:
		return fmt.Errorf("ip-map-output must be Y or N; file paths are no longer accepted")
	}
	return nil
}

func (v *mapExportValue) String() string {
	if *v {
		return "Y"
	}
	return "N"
}

func (*mapExportValue) Type() string { return "Y/N" }

func (o *redactOptions) flags(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&o.addresses, "redact-ips", nil, "Comma-separated literal instance IPs to replace in a separate collected-data copy (IPv4 and IPv6)")
	cmd.Flags().IntVarP(&o.workers, "redact-workers", "w", 4, "Maximum number of files processed concurrently during redaction (1-256)")
	cmd.Flags().BoolVar(&o.allIPs, "redact-all-ips", false, "Automatically discover and replace all recognized literal IPv4/IPv6 addresses in supported collected data")
	cmd.Flags().BoolVar(&o.logIPv4QuestionMark, "hide-log-ipv4", false, "Hide IPv4 in log bodies using ?; keep unique replacement IPs in paths/topology/metrics (requires --redact-all-ips)")
	cmd.Flags().BoolVar(&o.inPlace, "redact-in-place", false, "Replace collected files in place without a full second directory; original collected content is not retained")
	cmd.Flags().Var((*mapExportValue)(&o.mapOutput), "ip-map-output", "Y exports <collection-name>-ip-map.json beside the collection; N disables export (default); never upload the map")
	cmd.Flags().IntVar(&o.keepIPv4Octets, "keep-ipv4-octets", 0, "Keep the last IPv4 octet with prefix 10.0.0 (1), or the last two with prefix 198.18 (2); matching suffixes share replacement IPs")
	cmd.Flags().StringVar(&o.output, "redact-output", "", "New redacted directory outside the input; default: <collected-directory>-redacted")
	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("redact-workers") && o.workers == 0 {
			return fmt.Errorf("redact-workers must be between 1 and 256")
		}
		if cmd.Flags().Changed("redact-ips") && len(o.addresses) == 0 {
			return fmt.Errorf("redact-ips cannot be empty")
		}
		if cmd.Flags().Changed("redact-ips") && o.allIPs {
			return fmt.Errorf("redact-all-ips and redact-ips are mutually exclusive")
		}
		return o.validate()
	}
}

func (o *redactOptions) validate() error {
	if o.workers < 0 || o.workers > 256 {
		return fmt.Errorf("redact-workers must be between 1 and 256")
	}
	if o.keepIPv4Octets != 0 {
		if o.keepIPv4Octets != 1 && o.keepIPv4Octets != 2 {
			return fmt.Errorf("keep-ipv4-octets must be 1 or 2")
		}
		if o.logIPv4QuestionMark {
			return fmt.Errorf("keep-ipv4-octets and hide-log-ipv4 are mutually exclusive")
		}
		if !o.allIPs && len(o.addresses) == 0 {
			return fmt.Errorf("keep-ipv4-octets requires redact-ips or redact-all-ips")
		}
	}
	if o.mapOutput && !o.allIPs && len(o.addresses) == 0 {
		return fmt.Errorf("ip-map-output requires redact-ips or redact-all-ips")
	}
	if o.inPlace && o.output != "" {
		return fmt.Errorf("redact-in-place and redact-output are mutually exclusive")
	}
	if o.inPlace && !o.allIPs && len(o.addresses) == 0 {
		return fmt.Errorf("redact-in-place requires redact-ips or redact-all-ips")
	}
	if o.logIPv4QuestionMark && !o.allIPs {
		return fmt.Errorf("hide-log-ipv4 requires redact-all-ips")
	}
	if o.allIPs {
		if len(o.addresses) > 0 {
			return fmt.Errorf("redact-all-ips and redact-ips are mutually exclusive")
		}
		return nil
	}
	if len(o.addresses) == 0 {
		if o.output != "" {
			return fmt.Errorf("redact-output requires redact-ips or redact-all-ips")
		}
		return nil
	}
	_, err := redact.New(o.addresses)
	return err
}

func (o *redactOptions) afterCollect(input string, collectErr error) error {
	if collectErr != nil {
		return collectErr
	}
	if len(o.addresses) == 0 && !o.allIPs {
		return nil
	}
	return o.run(input)
}

func (o *redactOptions) run(input string) (runErr error) {
	if err := o.validate(); err != nil {
		return err
	}
	progress := newRedactionProgress()
	defer func() { progress.close(runErr) }()
	var r *redact.Redactor
	var err error
	workers := o.workers
	if workers == 0 {
		workers = 1
	}
	if o.allIPs {
		r, err = redact.DiscoverWithWorkers(input, progress.update, workers)
	} else {
		r, err = redact.New(o.addresses)
	}
	if err != nil {
		return err
	}
	if err := r.SetWorkers(workers); err != nil {
		return err
	}
	r.SetProgress(progress.update)
	if o.keepIPv4Octets != 0 {
		if err := r.KeepIPv4Octets(o.keepIPv4Octets); err != nil {
			return err
		}
	}
	if o.logIPv4QuestionMark {
		if err := r.MaskLogIPv4(); err != nil {
			return err
		}
	}
	if o.mapOutput {
		output := o.output
		if o.inPlace {
			output = input
		}
		absInput, err := filepath.Abs(input)
		if err != nil {
			return err
		}
		mapFile := filepath.Join(filepath.Dir(absInput), filepath.Base(absInput)+"-ip-map.json")
		if err := r.SaveMapping(mapFile, input, output); err != nil {
			return err
		}
		fmt.Printf("Private local IP map saved to %s. Do not upload this file.\n", mapFile)
	}
	var output string
	var report redact.Report
	if o.inPlace {
		fmt.Println("In-place redaction replaces the original collected content. Interrupted directories must not be packaged.")
		output, report, err = r.Rewrite(input)
	} else {
		output, report, err = r.Copy(input, o.output)
	}
	if err != nil {
		if o.inPlace {
			return err
		}
		return fmt.Errorf("IP redaction failed; no redacted output published; original collection remains local: %w", err)
	}
	if o.allIPs {
		fmt.Printf("Automatic IP redaction: %d unique IPs mapped.\n", report.IPs)
	}
	if o.logIPv4QuestionMark {
		fmt.Printf("Log body IPv4 replaced with ?: %d occurrences. Paths, topology and metrics retain unique replacement IPs.\n", report.LogIPv4QuestionMarks)
	}
	fmt.Printf("IP redaction complete: %d files, %d replacements.\nPackage ONLY this directory: %s\n", report.Files, report.Replacements, output)
	if o.inPlace {
		fmt.Println("Original collected content was replaced; no full original copy is retained. Server logs are unchanged.")
	} else {
		fmt.Println("The original collection still contains real IPs.")
	}
	return nil
}

func newRedactCmd() *cobra.Command {
	var options redactOptions
	cmd := &cobra.Command{
		Use:   "redact <collected-datadir> (--redact-ips <ip,ip,...> | --redact-all-ips)",
		Short: "Redact collected data before packaging; create a copy by default or explicitly rewrite in place",
		Args:  cobra.ExactArgs(1),
		// Redaction only reads local files. Do not inherit the root command's
		// TiUP repository initialization/close hooks or send telemetry.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			reportEnabled = false
			return nil
		},
		PersistentPostRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE:               func(cmd *cobra.Command, args []string) error { return options.run(args[0]) },
	}
	options.flags(cmd)
	return cmd
}
