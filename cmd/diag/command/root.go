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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/joomcode/errorx"
	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/pkg/config"
	"github.com/pingcap/diag/pkg/telemetry"
	"github.com/pingcap/diag/version"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/environment"
	tiupmeta "github.com/pingcap/tiup/pkg/environment"
	"github.com/pingcap/tiup/pkg/localdata"
	"github.com/pingcap/tiup/pkg/logger"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/repository"
	tiuptelemetry "github.com/pingcap/tiup/pkg/telemetry"
	"github.com/pingcap/tiup/pkg/tui"
	"github.com/pingcap/tiup/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	rootCmd       *cobra.Command
	gOpt          operator.Options
	skipConfirm   bool
	log           = logprinter.NewLogger("")
	reportEnabled bool // is telemetry report enabled
	teleReport    *telemetry.Report
	teleCommand   []string
	diagConfig    config.DiagConfig
)

func getParentNames(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}

	p := cmd.Parent()
	// always use 'diag' as the root command name
	if cmd.Parent() == nil {
		return []string{"diag"}
	}

	return append(getParentNames(p), cmd.Name())
}

func init() {
	logger.InitGlobalLogger()

	tui.AddColorFunctionsForCobra()
	tui.RegisterArg0("tiup diag")

	cobra.EnableCommandSorting = false

	nativeEnvVar := strings.ToLower(os.Getenv(localdata.EnvNameNativeSSHClient))
	if nativeEnvVar == "true" || nativeEnvVar == "1" || nativeEnvVar == "enable" {
		gOpt.NativeSSH = true
	}

	rootCmd = &cobra.Command{
		Use:           tui.OsArgs0(),
		Short:         "Collect metrics and information from a TiDB cluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.String(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// populate logger
			log.SetDisplayModeFromString(gOpt.DisplayMode)

			var err error
			var env *tiupmeta.Environment
			// unset component data dir to use clusters'
			os.Unsetenv(localdata.EnvNameComponentDataDir)
			if err = spec.Initialize("diag"); err != nil {
				return err
			}

			// Running in other OS/ARCH Should be fine we only download manifest file.
			env, err = tiupmeta.InitEnv(repository.Options{}, repository.MirrorOptions{})
			if err != nil {
				return err
			}
			tiupmeta.SetGlobalEnv(env)

			logger.EnableAuditLog(spec.AuditDir())

			teleCommand = getParentNames(cmd)

			if gOpt.NativeSSH {
				gOpt.SSHType = executor.SSHTypeSystem
				zap.L().Info("System ssh client will be used",
					zap.String(localdata.EnvNameNativeSSHClient, os.Getenv(localdata.EnvNameNativeSSHClient)))
				fmt.Println("The --native-ssh flag has been deprecated, please use --ssh=system")
			}
			diagConfig.Load()
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return tiupmeta.GlobalEnv().V1Repository().Mirror().Close()
		},
	}

	tui.BeautifyCobraUsageAndHelp(rootCmd)

	rootCmd.SetVersionTemplate(fmt.Sprintf("%s {{.Version}}\n", tui.OsArgs0()))

	rootCmd.PersistentFlags().Uint64Var(&gOpt.SSHTimeout, "ssh-timeout", 5, "Timeout in seconds to connect host via SSH, ignored for operations that don't need an SSH connection.")
	// the value of wait-timeout is also used for `systemctl` commands, as the default timeout of systemd for
	// start/stop operations is 90s, the default value of this argument is better be longer than that
	rootCmd.PersistentFlags().Uint64Var(&gOpt.OptTimeout, "wait-timeout", 180, "Timeout in seconds to wait for an operation to complete, ignored for operations that don't fit.")
	rootCmd.PersistentFlags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip all confirmations and assumes 'yes'")
	rootCmd.PersistentFlags().IntVarP(&gOpt.Concurrency, "concurrency", "c", 5, "max number of parallel tasks allowed")
	rootCmd.PersistentFlags().BoolVar(&gOpt.NativeSSH, "native-ssh", gOpt.NativeSSH, "(EXPERIMENTAL) Use the native SSH client installed on local system instead of the build-in one.")
	rootCmd.PersistentFlags().StringVar((*string)(&gOpt.SSHType), "ssh", "", "(EXPERIMENTAL) The executor type: 'builtin', 'system', 'none'.")
	rootCmd.PersistentFlags().StringVar(&gOpt.DisplayMode, "format", "default", "(EXPERIMENTAL) The format of output, available values are [default, json]")
	_ = rootCmd.PersistentFlags().MarkHidden("native-ssh")

	rootCmd.AddCommand(
		newCollectCmd(),
		newCollectDMCmd(),
		newCollectkCmd(),
		newPackageCmd(),
		newRebuildCmd(),
		newUploadCommand(),
		newHistoryCommand(),
		newCheckCmd(),
		newAuditCmd(),
		newConfigCmd(),
		newUtilCmd(),
	)
}

func printErrorMessageForNormalError(err error) {
	_, _ = tui.ColorErrorMsg.Fprintf(os.Stderr, "\nError: %s\n", err.Error())
}

func printErrorMessageForErrorX(err *errorx.Error) {
	msg := ""
	ident := 0
	causeErrX := err
	for causeErrX != nil {
		if ident > 0 {
			msg += strings.Repeat("  ", ident) + "caused by: "
		}
		currentErrMsg := causeErrX.Message()
		if len(currentErrMsg) > 0 {
			if ident == 0 {
				// Print error code only for top level error
				msg += fmt.Sprintf("%s (%s)\n", currentErrMsg, causeErrX.Type().FullName())
			} else {
				msg += fmt.Sprintf("%s\n", currentErrMsg)
			}
			ident++
		}
		cause := causeErrX.Cause()
		if c := errorx.Cast(cause); c != nil {
			causeErrX = c
		} else {
			if cause != nil {
				if ident > 0 {
					// The error may have empty message. In this case we treat it as a transparent error.
					// Thus `ident == 0` can be possible.
					msg += strings.Repeat("  ", ident) + "caused by: "
				}
				msg += fmt.Sprintf("%s\n", cause.Error())
			}
			break
		}
	}
	_, _ = tui.ColorErrorMsg.Fprintf(os.Stderr, "\nError: %s", msg)
}

func extractSuggestionFromErrorX(err *errorx.Error) string {
	cause := err
	for cause != nil {
		v, ok := cause.Property(utils.ErrPropSuggestion)
		if ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		cause = errorx.Cast(cause.Cause())
	}

	return ""
}

// Execute executes the root command
func Execute() {
	zap.L().Info("Execute command", zap.String("command", tui.OsArgs()))
	zap.L().Debug("Environment variables", zap.Strings("env", os.Environ()))

	teleReport = new(telemetry.Report)
	reportEnabled = tiuptelemetry.Enabled()
	if reportEnabled {
		eventUUID := os.Getenv(localdata.EnvNameTelemetryEventUUID)
		if eventUUID == "" {
			eventUUID = uuid.New().String()
		}
		teleReport.UUID = eventUUID
		teleReport.Version = telemetry.GetVersion()
	}

	start := time.Now()
	code := 0
	err := rootCmd.Execute()
	if err != nil {
		code = 1
	}

	zap.L().Info("Execute command finished", zap.Int("code", code), zap.Error(err))

	if reportEnabled {
		f := func() {
			defer func() {
				if r := recover(); r != nil {
					if environment.DebugMode {
						log.Debugf("Recovered in telemetry report: %v", r)
					}
				}
			}()

			teleReport.ExitCode = int32(code)
			teleReport.ExecutionTime = uint64(time.Since(start).Milliseconds())
			teleReport.Command = strings.Join(teleCommand, " ")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			tele := telemetry.NewTelemetry()
			err := tele.Report(ctx, teleReport)
			if environment.DebugMode {
				if err != nil {
					log.Infof("report failed: %v", err)
				}
				if data, err := json.Marshal(teleReport); err == nil {
					log.Debugf("report: %s\n", string(data))
					fmt.Println(string(data))
				}
			}
			cancel()
		}

		f()
	}

	if err != nil {
		switch strings.ToLower(gOpt.DisplayMode) {
		case "json":
			obj := struct {
				Err string `json:"error"`
			}{
				Err: err.Error(),
			}
			data, err := json.Marshal(obj)
			if err != nil {
				fmt.Printf("{\"error\": \"%s\"}", err)
				break
			}
			fmt.Fprintln(os.Stderr, string(data))
		default:
			if errx := errorx.Cast(err); errx != nil {
				printErrorMessageForErrorX(errx)
			} else {
				printErrorMessageForNormalError(err)
			}

			if !errorx.HasTrait(err, utils.ErrTraitPreCheck) {
				logger.OutputDebugLog("tiup-diag")
			}

			if errx := errorx.Cast(err); errx != nil {
				if suggestion := extractSuggestionFromErrorX(errx); len(suggestion) > 0 {
					_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", suggestion)
				}
			}
		}
	}

	err = logger.OutputAuditLogIfEnabled()
	if err != nil {
		zap.L().Warn("Write audit log file failed", zap.Error(err))
		code = 1
	}

	color.Unset()

	if code != 0 {
		os.Exit(code)
	}
}

func scrubClusterName(n string) string {
	// prepend the telemetry secret to cluster name, so that two installations
	// of tiup with the same cluster name produce different hashes
	return "cluster_" + tiuptelemetry.SaltedHash(n)
}
