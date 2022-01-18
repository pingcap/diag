package command

import (
	"context"

	"github.com/pingcap/diag/pkg/packager"
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

			if opt.UserName == "" || opt.Password == "" {
				opt.UserName, opt.Password = packager.Credentials()
			}
			opt.Client = packager.InitClient(opt.Endpoint)
			opt.Concurrency = gOpt.Concurrency

			ctx := context.WithValue(
				context.Background(),
				logprinter.ContextKeyLogger,
				log,
			)
			_, err := packager.Upload(ctx, &opt, skipConfirm)
			return err
		},
	}

	cmd.Flags().StringVarP(&opt.Alias, "alias", "", "", "the Alias of upload file.")
	cmd.Flags().StringVarP(&opt.Endpoint, "endpoint", "", "https://clinic.pingcap.com:4433", "the clinic service Endpoint.")
	cmd.Flags().StringVarP(&opt.Issue, "issue", "", "", "related jira oncall Issue, example: ONCALL-1131")
	cmd.Flags().StringVarP(&opt.UserName, "username", "u", "", "username of clinic service")
	cmd.Flags().StringVarP(&opt.Password, "password", "p", "", "password of clinic service")

	return cmd
}
