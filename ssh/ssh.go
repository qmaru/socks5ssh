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

	if client.config.AuthType == 1 {
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
	} else if client.config.AuthType == 2 {
		conf.Auth = []ssh.AuthMethod{
			ssh.Password(client.config.AuthData),
		}
	} else {
		return nil, fmt.Errorf("auth type error")
	}

	sshClient, err := ssh.Dial("tcp", client.config.Address, conf)
	if err != nil {
		return nil, err
	}
	return sshClient, nil
}
