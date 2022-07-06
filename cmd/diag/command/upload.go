package command

import (
	"context"
	"os"
	"strings"

	"github.com/pingcap/diag/pkg/packager"
	"github.com/pingcap/diag/pkg/telemetry"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/spf13/cobra"
)

func newUploadCommand() *cobra.Command {
	opt := packager.UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "upload a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			opt.FilePath = args[0]

			var saveconfig bool

			// get endpoint from config file
			if diagConfig.Clinic.Region == "" {
				err := diagConfig.InteractiveSetRegion()
				if err != nil {
					return err
				}
				saveconfig = true

			}
			opt.Cert = diagConfig.Clinic.Region.Cert()
			if opt.Endpoint == "" {
				opt.Endpoint = diagConfig.Clinic.Region.Endpoint()
			}
			opt.Endpoint = strings.Trim(opt.Endpoint, "/")

			// get token from env or config file
			opt.Token = os.Getenv("CLINIC_TOKEN")
			if opt.Token == "" {
				if diagConfig.Clinic.Token == "" {
					diagConfig.InteractiveSetToken()
					saveconfig = true
				}
				opt.Token = diagConfig.Clinic.Token
			}

			if saveconfig {
				err := diagConfig.Save()
				if err != nil {
					return err
				}
			}

			opt.Client = packager.InitClient(opt.Endpoint)
			opt.Concurrency = gOpt.Concurrency

			if reportEnabled {
				teleReport.CommandInfo = &telemetry.UploadInfo{
					Endpoint: opt.Endpoint,
				}
			}

			ctx := context.WithValue(
				context.Background(),
				logprinter.ContextKeyLogger,
				log,
			)
			_, err := packager.Upload(ctx, &opt, skipConfirm)

			// TODO: add size info for upload (similar with `package`)
			// if reportEnabled {}

			return err
		},
	}

	cmd.Flags().StringVar(&opt.Alias, "alias", "", "the Alias of upload file.")
	cmd.Flags().StringVar(&opt.Endpoint, "endpoint", "", "the clinic service Endpoint.")
	cmd.Flags().StringVar(&opt.Issue, "issue", "", "related jira oncall Issue, example: ONCALL-1131")
	cmd.Flags().BoolVar(&opt.Rebuild, "rebuild", true, "rebuild package immediately after upload")

	cmd.Flags().MarkHidden("endpoint")

	return cmd
}
