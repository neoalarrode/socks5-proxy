# SOCKS5 Proxy

Lightweight SOCKS5 proxy server for Linux and Windows. Single binary, zero dependencies.

## Features

- Full SOCKS5 protocol (RFC 1928)
- Username/password authentication (RFC 1929)
- IPv4, IPv6 and domain name support
- ~2 MB standalone binary
- Cross-platform: Linux and Windows (amd64)

## Download

Grab the latest binaries from [Releases](https://github.com/neoalarrode/socks5-proxy/releases).

## Build from source

```bash
git clone https://github.com/neoalarrode/socks5-proxy.git
cd socks5-proxy
bash build.sh
```

Binaries are placed in `dist/`.

## Usage

```bash
# Open proxy (no authentication)
./socks5-proxy-linux-amd64

# With authentication
./socks5-proxy-linux-amd64 -user myuser -pass mypassword

# Custom port and log file
./socks5-proxy-linux-amd64 -port 9050 -user admin -pass secret -log proxy.log
```

### Windows

```powershell
socks5-proxy-windows-amd64.exe -user myuser -pass mypassword
```

### Options

| Flag    | Default   | Description                            |
|---------|-----------|----------------------------------------|
| `-host` | `0.0.0.0` | Listen address                         |
| `-port` | `1080`    | Listen port                            |
| `-user` | *(empty)* | Username (enables authentication)      |
| `-pass` | *(empty)* | Password (required when -user is set)  |
| `-log`  | *(empty)* | Log file path (default: stderr)        |

### Test with curl

```bash
curl --socks5 127.0.0.1:1080 --proxy-user user:pass http://httpbin.org/ip
```

## License

Proprietary. See [LICENSE](LICENSE).
