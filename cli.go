package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/xuperchain/xuperchain/service/pb"

)

var (
	buildVersion = ""
	buildDate    = ""
	commitHash   = ""
)

func NewCli() *Cli {
	rootCmd := &cobra.Command{
		Use:           "xchain-cli",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       xchainVersion(),
	}
	return &Cli{
		rootCmd: rootCmd,
	}
}
// CommandFunc 代表了一个子命令，用于往Cli注册子命令
type CommandFunc func(c *Cli) *cobra.Command
var (
	// commands 用于收集所有的子命令，在启动的时候统一往Cli注册
	commands []CommandFunc
)


// Cli 是所有子命令执行的上下文.
type Cli struct {

	rootCmd *cobra.Command
	xclient pb.XchainClient

	eventClient pb.EventServiceClient
}

// EventClient get EventService client
func (c *Cli) EventClient() pb.EventServiceClient {
	return c.eventClient
}

func xchainVersion() string {
	return fmt.Sprintf("%s-%s %s", buildVersion, commitHash, buildDate)
}

