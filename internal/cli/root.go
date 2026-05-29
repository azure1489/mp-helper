// Package cli 实现 mp-cli 的 cobra 命令。
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/azure1489/mp-helper/internal/client"
)

func newClient() (*client.Client, error) {
	base := os.Getenv("MP_HELPER_API_URL")
	if base == "" {
		return nil, fmt.Errorf("MP_HELPER_API_URL is not set")
	}
	return client.New(base, os.Getenv("MP_HELPER_API_KEY")), nil
}

func printJSON(cmd *cobra.Command, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "mp-cli",
		Short:         "mp-helper CLI：上传微信公众号素材、创建草稿",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newHealthCmd(), newMaterialCmd(), newDraftCmd())
	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
