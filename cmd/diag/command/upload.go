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

			userName, password := packager.Credentials()
			opt.UserName = userName
			opt.Password = password
			opt.Client = packager.InitClient(opt.Endpoint)

			ctx := context.WithValue(
				context.Background(),
				logprinter.ContextKeyLogger,
				log,
			)
			_, err := packager.Upload(ctx, &opt)
			return err
		},
	}

	cmd.Flags().StringVarP(&opt.Alias, "Alias", "", "", "the Alias of upload file.")
	cmd.Flags().StringVarP(&opt.Endpoint, "Endpoint", "", "https://clinic.pingcap.com:4433", "the clinic service Endpoint.")
	cmd.Flags().StringVarP(&opt.Issue, "Issue", "", "", "related jira oncall Issue, example: ONCALL-1131")

	return cmd
}
