package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"socks5ssh/proxy"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	localAddress string
	sshAddress   string
	sshUser      string
	sshPass      bool
	sshKey       string
	rootCmd      = &cobra.Command{
		Use:     "socks5ssh",
		Short:   "Proxy Over SSH By Socks5/HTTP",
		Version: "1.2-230106",
		Run: func(cmd *cobra.Command, args []string) {
			err := proxy.AddressChecker(sshAddress)
			if err != nil {
				log.Fatalf("%s - %s", err.Error(), sshAddress)
			}

			protocal, listenAddr := ProtocolRouter(localAddress)

			err = proxy.AddressChecker(listenAddr)
			if err != nil {
				log.Fatalf("%s - %s", err.Error(), listenAddr)
			}

			var authData string
			var authType int

			if sshKey != "" {
				authData = sshKey
				authType = 1
				log.Println("[Method] Private Key")
			} else {
				if sshPass {
					fmt.Printf("Enter password: ")
					password, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						log.Fatal(err)
					}
					authData = string(password)
					authType = 2
					fmt.Println("")
					log.Println("[Method] Password")
				} else {
					log.Fatal("auth method is required (-p or -k)")
				}
			}

			tunnel := new(proxy.SSHProxy)
			if protocal == "http" {
				log.Printf("[Remote] server: %s\n", sshAddress)
				log.Printf("[Local] %s\n", localAddress)
				log.Println("[State] running...")
				err := tunnel.HTTPRun(listenAddr, sshAddress, sshUser, authData, authType)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Printf("[Remote] server: %s\n", sshAddress)
				log.Printf("[Local] socks5://%s\n", localAddress)
				log.Println("[State] connecting...")

				go func() {
					err := tunnel.Socks5Run(listenAddr, sshAddress, sshUser, authData, authType)
					if err != nil {
						log.Fatal(err)
					}
				}()

				go func() {
					for {
						result := proxy.RunningCheck(listenAddr)
						if result {
							log.Println("[State] running...")
							break
						}
					}
				}()

				select {}
			}
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&localAddress, "local", "l", "", "Local Socks5/HTTP Listen Address <host>:<port>")
	rootCmd.Flags().StringVarP(&sshAddress, "remote", "r", "", "Remote SSH Address <host>:<port>")
	rootCmd.Flags().StringVarP(&sshUser, "user", "u", "", "Remote SSH Username")
	rootCmd.Flags().BoolVarP(&sshPass, "password", "p", false, "Remote SSH Password")
	rootCmd.Flags().StringVarP(&sshKey, "key", "k", "", "Remote SSH Private Key")
	rootCmd.MarkFlagRequired("local")
	rootCmd.MarkFlagRequired("remote")
	rootCmd.MarkFlagRequired("user")
}

func Execute() {
	rootCmd.DisableFlagsInUseLine = true
	rootCmd.AddCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ProtocolRouter(localAddress string) (string, string) {
	var protocal string
	var listenAddr string
	if strings.Contains(localAddress, "http://") {
		protocal = "http"
		listenAddr = strings.ReplaceAll(localAddress, "http://", "")
	} else {
		protocal = "socks5"
		listenAddr = localAddress
	}
	return protocal, listenAddr
}
