package ssh

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config ssh config
//
//	AuthType: 1. Private key 2. Password
type Config struct {
	Address  string
	User     string
	AuthData string
	AuthType int
	Timeout  int
}

type Client struct {
	config *Config
}

func New(conf *Config) *Client {
	if conf.Timeout == 0 {
		conf.Timeout = 15
	}

	server := &Client{
		config: conf,
	}
	return server
}

// Client Create a ssh client
func (client *Client) Connect() (*ssh.Client, error) {
	conf := &ssh.ClientConfig{
		Timeout:         time.Duration(client.config.Timeout) * time.Second,
		User:            client.config.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	switch client.config.AuthType {
	case 1:
		key, err := os.ReadFile(client.config.AuthData)
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		conf.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	case 2:
		conf.Auth = []ssh.AuthMethod{
			ssh.Password(client.config.AuthData),
		}
	default:
		return nil, fmt.Errorf("auth type error")
	}

	sshClient, err := ssh.Dial("tcp", client.config.Address, conf)
	if err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, _, err := sshClient.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return
			}
		}
	}()

	return sshClient, nil
}
