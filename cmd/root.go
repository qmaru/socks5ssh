package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"socks5ssh/tunnel"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const VERSION = "YOURVERSION"

var (
	debug        bool
	localAddress string
	sshAddress   string
	sshUser      string
	sshPass      bool
	sshKey       string
	dns          string
	rootCmd      = &cobra.Command{
		Use:     "socks5ssh",
		Short:   "Use socks5 or http to connect ssh tunnel to forward data",
		Version: VERSION,
		Example: "socks5ssh -r remote.example.com:22 -l 127.0.0.1:1080 -u root -p",
		Run: func(cmd *cobra.Command, args []string) {
			err := tunnel.AddressChecker(sshAddress)
			if err != nil {
				log.Fatalf("%s - %s", err.Error(), sshAddress)
			}

			protocal, listenAddr := ProtocolRouter(localAddress)

			err = tunnel.AddressChecker(listenAddr)
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

			tun := new(tunnel.Tunnel)
			tun.LocalAddress = listenAddr
			tun.RemoteAddress = sshAddress
			tun.RemoteUser = sshUser
			tun.RemoteAuthData = authData
			tun.RemoteAuthType = authType
			tun.DNSServer = dns
			tun.Debug = debug

			if debug {
				log.Printf("[Debug] %v\n", debug)
			}

			if protocal == "http" {
				log.Printf("[Remote] server: %s\n", sshAddress)
				log.Printf("[Local] %s\n", localAddress)
				log.Println("[State] running...")
				err := tun.HTTPRun()
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Printf("[Remote] server: %s\n", sshAddress)
				log.Printf("[Local] socks5://%s\n", localAddress)
				log.Printf("[Dns] %s\n", dns)
				log.Println("[State] connecting...")

				go func() {
					err := tun.Socks5Run()
					if err != nil {
						log.Fatal(err)
					}
				}()

				go func() {
					ticker := time.NewTicker(100 * time.Millisecond)
					defer ticker.Stop()

					for range ticker.C {
						result := tunnel.RunningCheck(listenAddr)
						if result {
							log.Println("[State] running...")
							return
						}
					}
				}()

				select {}
			}
		},
	}
)

func init() {
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Debug mode")
	rootCmd.Flags().StringVarP(&localAddress, "local", "l", "", "Local Socks5/HTTP Listen Address <host>:<port>")
	rootCmd.Flags().StringVarP(&sshAddress, "remote", "r", "", "Remote SSH Address <host>:<port>")
	rootCmd.Flags().StringVarP(&sshUser, "user", "u", "", "Remote SSH Username")
	rootCmd.Flags().BoolVarP(&sshPass, "password", "p", false, "Remote SSH Password")
	rootCmd.Flags().StringVarP(&sshKey, "key", "k", "", "Remote SSH Private Key")
	rootCmd.Flags().StringVarP(&dns, "dns", "d", "8.8.8.8", `DNS Resolver [tcp,udp,dot]`)
	rootCmd.MarkFlagRequired("local")
	rootCmd.MarkFlagRequired("remote")
	rootCmd.MarkFlagRequired("user")
}

func Execute() {
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
