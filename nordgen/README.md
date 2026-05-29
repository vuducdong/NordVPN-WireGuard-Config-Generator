# NordVPN WireGuard Config Generator

A fast, asynchronous command-line tool that fetches the live NordVPN server catalogue and generates ready-to-use WireGuard configuration files—one per server, organized by group, country, and city. The lowest-latency endpoint per location is automatically surfaced into a separate `best_configs/` tree.

## Features

- **Complete catalogue:** Generates a `.conf` for every WireGuard-enabled NordVPN server (typically ~7,500 active configurations).
- **Optimized subset:** Selects the lowest-load server per city into `best_configs/` for quick selection.
- **Distance-aware:** Ranks servers by load and Haversine distance from your detected geolocation.
- **Advanced filtering:** Selectively generate configurations for specific server groups (Standard, P2P, Onion, Double, Dedicated) and explicitly exclude dedicated IPs.
- **Concurrent I/O:** Non-blocking HTTP and batched filesystem writes; a full run completes in seconds.
- **Zero stored credentials:** Your access token is used in-memory only and is never written to disk.
- **Customizable:** DNS, endpoint mode (hostname or IP), and keepalive intervals are configurable per run.

## Requirements

- Python 3.11 or later
- An active NordVPN subscription and a [personal access token](https://my.nordaccount.com/dashboard/nordvpn/access-tokens/)

## Installation

The recommended installer for command-line applications is [`pipx`](https://pipx.pypa.io/), which isolates the tool in its own environment:

```bash
pipx install nord-config-generator
```

Plain `pip` works equally well:

```bash
pip install nord-config-generator
```

## Usage

### Generate configurations

Interactive mode prompts for your token and preferences:

```bash
nordgen
```

*(Note: The explicit `nordgen generate` command is also supported.)*

Non-interactive mode (fully scripted):

```bash
nordgen --token <YOUR_TOKEN> --dns 1.1.1.1 --keepalive 15 --group standard p2p --exclude-dedicated
```

| Flag | Description | Default |
|---|---|---|
| `-t`, `--token` | NordVPN access token (64-character hex) | Prompted |
| `-d`, `--dns` | DNS server written into each config | `103.86.96.100` |
| `-i`, `--ip` | Use IP addresses instead of hostnames for `Endpoint` | Hostname |
| `-k`, `--keepalive` | `PersistentKeepalive` value in seconds | `25` |
| `-g`, `--group` | Server groups to include (`standard`, `p2p`, `dedicated`, `onion`, `double`) | All groups |
| `-e`, `--exclude-dedicated`| Exclude servers in the dedicated IP group | `False` |

> **Note on Dedicated IP Servers:** 
> Connecting to servers within the `dedicated` group requires an active **Dedicated IP add-on** purchased in addition to your standard NordVPN subscription. If your account does not include this add-on, configurations generated for dedicated servers will successfully generate but will fail to connect. If you do not own this add-on, it is highly recommended to use the `-e` (`--exclude-dedicated`) flag to omit them from your output.

### Retrieve the NordLynx private key

If you only need the raw private key for manual use:

```bash
nordgen get-key -t <YOUR_TOKEN>
```

## Output

Each run creates a timestamped directory in the current working directory. The configurations are sorted into standard and optimal trees, further categorized by the server's group combination:

```text
nordvpn_configs_20260419_143022/
├── configs/
│   └── <group_combo>/<country>/<city>/<server_name>.conf
└── best_configs/
    └── <group_combo>/<country>/<city>/<server_name>.conf
```

Use any `.conf` file directly with the [WireGuard client](https://www.wireguard.com/install/) on Windows, macOS, Linux, iOS, or Android.

## Security

- The access token is read into volatile memory and discarded at process exit.
- Token input is masked in the terminal and bypasses shell history.
- The generated `[Interface]` block contains your private key—treat the output directory as sensitive and store it accordingly.

## License

GPL-3.0-or-later. See [LICENSE](LICENSE) for full text.

## Links

- **Source:** [github.com/mustafachyi/NordVPN-WireGuard-Config-Generator](https://github.com/mustafachyi/NordVPN-WireGuard-Config-Generator)
- **Issues:** [Bug Tracker](https://github.com/mustafachyi/NordVPN-WireGuard-Config-Generator/issues)