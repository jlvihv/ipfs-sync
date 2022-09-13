package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var (
	masterPort int
	masterAddr string
	host       string
	port       int
)

var rootCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(masterCmd)
	rootCmd.AddCommand(slaveCmd)

	masterCmd.Flags().IntVarP(&masterPort, "port", "p", 8888, "master port")
	slaveCmd.Flags().StringVarP(&masterAddr, "master", "m", "localhost:8888", "master address")
	slaveCmd.Flags().StringVarP(&host, "host", "H", "localhost", "slave host")
	slaveCmd.Flags().IntVarP(&port, "port", "p", 10000, "slave port")
}
