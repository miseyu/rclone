package rmdir

import (
	"github.com/miseyu/rclone/cmd"
	"github.com/miseyu/rclone/fs"
	"github.com/spf13/cobra"
)

func init() {
	cmd.Root.AddCommand(rmdirsCmd)
}

var rmdirsCmd = &cobra.Command{
	Use:   "rmdirs remote:path",
	Short: `Remove empty directories under the path.`,
	Long: `This removes any empty directories (or directories that only contain
empty directories) under the path that it finds, including the path if
it has nothing in.

This is useful for tidying up remotes that rclone has left a lot of
empty directories in.

`,
	Run: func(command *cobra.Command, args []string) {
		cmd.CheckArgs(1, 1, command, args)
		fdst := cmd.NewFsDst(args)
		cmd.Run(true, false, command, func() error {
			return fs.Rmdirs(fdst, "")
		})
	},
}
