# socks5ssh

forward ssh using socks5 or http.

## 命令参数

```shell
Proxy Over SSH By Socks5/HTTP

Usage:
  socks5ssh

Flags:
  -h, --help            help for socks5ssh
  -k, --key string      Remote SSH Private Key
  -l, --local string    Local Socks5/HTTP Listen Address <host>:<port>
  -p, --password        Remote SSH Password
  -r, --remote string   Remote SSH Address <host>:<port>
  -u, --user string     Remote SSH Username
  -v, --version         version for socks5ssh
```

## 使用

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
