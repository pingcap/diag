package command

import (
	"context"

	"github.com/pingcap/diag/pkg/packager"
	"github.com/pingcap/errors"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/spf13/cobra"
)

func newDownloadCommand() *cobra.Command {
	opt := packager.DownloadOptions{}
	cmd := &cobra.Command{
		Use:   "download --uuid=<uuid>|--alias=<alias>|--cluster=<ClusterID>|<url>",
		Short: "download file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && opt.Endpoint == "" {
				return cmd.Help()
			}

			if len(args) >= 1 {
				if err := packager.ParseURL(&opt, args[0]); err != nil {
					return err
				}
			}

			userName, password := packager.Credentials()
			opt.UserName = userName
			opt.Password = password
			opt.Client = packager.InitClient(opt.Endpoint)

			ctx := context.WithValue(
				context.Background(),
				logprinter.ContextKeyLogger,
				log,
			)
			if opt.FileUUID != "" {
				return packager.Download(ctx, &opt)
			}

			if opt.FileAlias != "" {
				return packager.DownloadFilesByAlias(ctx, &opt)
			}

			if opt.ClusterID > 0 {
				return packager.DownloadFilesByClusterID(ctx, &opt)
			}

			return errors.New("unsupport parameter")
		},
	}

	cmd.Flags().StringVarP(&opt.FileUUID, "uuid", "", "", "the uuid of file")
	cmd.Flags().StringVarP(&opt.FileAlias, "alias", "", "", "the alias of file")
	cmd.Flags().Uint64VarP(&opt.ClusterID, "cluster-id", "", 0, "the cluster id of file")
	cmd.Flags().StringVarP(&opt.Endpoint, "endpoint", "", "", "the clinic service Endpoint.")

	return cmd
}
