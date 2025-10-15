package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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

			protocol, listenAddr := ProtocolRouter(localAddress)

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
					var password []byte
					var err error

					if term.IsTerminal(int(syscall.Stdin)) {
						fmt.Printf("Enter password: ")
						password, err = term.ReadPassword(int(syscall.Stdin))
						if err != nil {
							log.Fatal(err)
						}
						fmt.Println("")
					} else {
						envPassword := os.Getenv("SSH_PASSWORD")
						if envPassword == "" {
							log.Fatal("password is required: use interactive terminal or set SSH_PASSWORD environment variable")
						}
						password = []byte(envPassword)
						log.Println("[Info] using password from environment variable")
					}

					if len(password) == 0 {
						log.Fatal("password cannot be empty")
					}

					authData = string(password)
					authType = 2
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

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				log.Println("\n[State] shutting down...")
				if err := tun.Close(); err != nil {
					log.Printf("[Error] close tunnel: %v\n", err)
				}
				os.Exit(0)
			}()

			if debug {
				log.Printf("[Debug] %v\n", debug)
			}

			switch protocol {
			case "http":
				log.Printf("[Remote] server: %s\n", sshAddress)
				log.Printf("[Local] http://%s\n", listenAddr)
				log.Println("[State] running...")
				err := tun.HTTPRun()
				if err != nil {
					log.Fatal(err)
				}
			default:
				log.Printf("[Remote] server: %s\n", sshAddress)
				log.Printf("[Local] socks5://%s\n", listenAddr)
				log.Printf("[Dns] %s\n", dns)
				log.Println("[State] connecting...")

				errChan := make(chan error, 1)

				go func() {
					err := tun.Socks5Run()
					if err != nil {
						errChan <- err
					}
				}()

				go func() {
					ticker := time.NewTicker(100 * time.Millisecond)
					defer ticker.Stop()

					timeout := time.After(30 * time.Second)

					for {
						select {
						case <-ticker.C:
							result := tunnel.RunningCheck(listenAddr)
							if result {
								log.Println("[State] running...")
								return
							}
						case <-timeout:
							log.Println("[Warning] connection check timeout")
							return
						}
					}
				}()

				if err := <-errChan; err != nil {
					log.Fatal(err)
				}

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
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ProtocolRouter(localAddress string) (string, string) {
	var protocol string
	var listenAddr string

	localAddress = strings.TrimSpace(localAddress)

	if strings.HasPrefix(localAddress, "http://") {
		protocol = "http"
		listenAddr = strings.TrimPrefix(localAddress, "http://")
	} else if strings.HasPrefix(localAddress, "socks5://") {
		protocol = "socks5"
		listenAddr = strings.TrimPrefix(localAddress, "socks5://")
	} else {
		protocol = "socks5"
		listenAddr = localAddress
	}

	return protocol, listenAddr
}
