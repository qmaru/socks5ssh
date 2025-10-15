# socks5ssh

Use socks5 or http to connect ssh tunnel to forward data.

## Command

```shell
Usage:
  socks5ssh [flags]

Examples:
socks5ssh -r remote.example.com:22 -l 127.0.0.1:1080 -u root -p

Flags:
      --debug           Debug mode
  -d, --dns string      DNS Resolver [tcp,udp,dot] (default "8.8.8.8")
  -h, --help            help for socks5ssh
  -k, --key string      Remote SSH Private Key
  -l, --local string    Local Socks5/HTTP Listen Address <host>:<port>
  -p, --password        Remote SSH Password
  -r, --remote string   Remote SSH Address <host>:<port>
  -u, --user string     Remote SSH Username
  -v, --version         version for socks5ssh
```

## Usage

### socks5

```shell
# password
socks5ssh -l 127.0.0.1:1080 -r ssh_server:ssh_port -u ssh_user -p

# key
socks5ssh -l 127.0.0.1:1080 -r ssh_server:ssh_port -u ssh_user -k ~/.ssh/id_rsa
```

### http

```shell
# password
socks5ssh -l http://127.0.0.1:1080 -r ssh_server:ssh_port -u ssh_user -p

# key
socks5ssh -l http://127.0.0.1:1080 -r ssh_server:ssh_port -u ssh_user -k ~/.ssh/id_rsa
```

## Case

```shell
# Linux
socks5ssh -l 127.0.0.1:1080 -r ssh_server:ssh_port -u ssh_user -p

export http_proxy="socks5://127.0.0.1:1080"
export https_proxy="socks5://127.0.0.1:1080"

curl ip.sb
```

## Docker

```shell
# host
docker run --rm --net=host -e SSH_PASSWORD='123456' ghcr.io/qmaru/socks5ssh -r remote.example.com:22 -l 127.0.0.1:1080 -u root -p --debug

# nat
docker run --rm -p 1080:1080 -e SSH_PASSWORD='123456' ghcr.io/qmaru/socks5ssh -r remote.example.com:22 -l 0.0.0.0:1080 -u root -p --debug
```
