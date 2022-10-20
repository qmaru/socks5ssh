package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"socks5ssh/proxy"

	"github.com/spf13/cobra"
)

var (
	localAddress string
	sshAddress   string
	sshUser      string
	sshPass      string
	rootCmd      = &cobra.Command{
		Use:     "socks5ssh",
		Short:   "Proxy Over SSH By Socks5/HTTP",
		Version: "1.0-221020",
		Run: func(cmd *cobra.Command, args []string) {
			if result, ok := proxy.AddressChecker(sshAddress); ok {
				tunnel := new(proxy.SSHProxy)
				if protocal, address := ProtocolRouter(localAddress); protocal == "http" {
					log.Printf("[Remote] Server: %s\n", sshAddress)
					log.Printf("[Local] %s\n", localAddress)
					err := tunnel.HTTPRun(address, sshAddress, sshUser, sshPass)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					log.Printf("[Remote] Server: %s\n", sshAddress)
					log.Printf("[Local] socks5://%s\n", localAddress)
					err := tunnel.Socks5Run(address, sshAddress, sshUser, sshPass)
					if err != nil {
						log.Fatal(err)
					}
				}
			} else {
				log.Println(result)
			}
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&localAddress, "local", "l", "", "Local Socks5/HTTP Listen Address <host>:<port>")
	rootCmd.Flags().StringVarP(&sshAddress, "remote", "r", "", "Remote SSH Address <host>:<port>")
	rootCmd.Flags().StringVarP(&sshUser, "user", "u", "", "Remote SSH Username")
	rootCmd.Flags().StringVarP(&sshPass, "password", "p", "", "Remote SSH Password")
	rootCmd.MarkFlagRequired("local")
	rootCmd.MarkFlagRequired("remote")
	rootCmd.MarkFlagRequired("user")
	rootCmd.MarkFlagRequired("password")
}

func Execute() {
	rootCmd.DisableFlagsInUseLine = true
	rootCmd.AddCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ProtocolRouter(localAddress string) (protocal, newAddress string) {
	if strings.Contains(localAddress, "http://") {
		protocal = "http"
		newAddress = strings.ReplaceAll(localAddress, "http://", "")
	} else {
		protocal = "socks5"
		newAddress = localAddress
	}

	if result, ok := proxy.AddressChecker(newAddress); ok {
		return
	} else {
		log.Fatal(result)
	}
	return
}
