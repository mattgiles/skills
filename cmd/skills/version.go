package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattgiles/skills/internal/ui"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, _ []string) {
			view := ui.New(cmd)
			view.Header("Version")
			_ = view.KeyValues("Build", [][2]string{
				{"Version", version},
				{"Commit", commit},
				{"Date", date},
				{"Go", runtime.Version()},
				{"Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
			})
		},
	}
}

func versionSummary() string {
	parts := []string{
		fmt.Sprintf("version=%s", version),
		fmt.Sprintf("commit=%s", commit),
		fmt.Sprintf("date=%s", date),
		fmt.Sprintf("go=%s", runtime.Version()),
		fmt.Sprintf("platform=%s/%s", runtime.GOOS, runtime.GOARCH),
	}
	return strings.Join(parts, " ")
}
