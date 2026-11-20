package ssh

import (
	"fmt"
	"log"
	"os"
	"sync"
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
	Debug    bool
}

type Client struct {
	config *Config
	client *ssh.Client
	mu     sync.Mutex
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
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.client != nil {
		return client.client, nil
	}

	if client.config.Debug {
		log.Printf("[SSH] connecting user=%s server=%s\n", client.config.User, client.config.Address)
	}
	sshClient, err := client.createConnection()
	if err != nil {
		if client.config.Debug {
			log.Printf("[SSH] connect failed server=%s error=%v\n", client.config.Address, err)
		}
		return nil, err
	}
	if client.config.Debug {
		log.Printf("[SSH] connected server=%s\n", client.config.Address)
	}
	return sshClient, nil
}

func (client *Client) createConnection() (*ssh.Client, error) {
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

	client.client = sshClient
	go client.keepAlive(sshClient)

	return sshClient, nil
}

func (client *Client) keepAlive(sshClient *ssh.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		client.mu.Lock()
		if client.client != sshClient {
			client.mu.Unlock()
			return
		}
		client.mu.Unlock()

		_, _, err := sshClient.SendRequest("keepalive@openssh.com", true, nil)
		if err != nil {
			if client.config.Debug {
				log.Printf("[SSH] keepalive failed server=%s error=%v\n", client.config.Address, err)
			}
			client.mu.Lock()
			if client.client == sshClient {
				client.client = nil
			}
			client.mu.Unlock()
			_ = sshClient.Close()
			return
		}
	}
}

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.client != nil {
		err := client.client.Close()
		client.client = nil
		return err
	}
	return nil
}
