# A2S-Cache: Source Engine Query Cache in Go

A lightweight, high-performance caching server for [A2S queries](https://developer.valvesoftware.com/wiki/Server_queries) used by Source Engine games. Written in Go, this project provides a memory-efficient alternative to solutions like [SourceEngineQueryCacher](https://github.com/hyperxpro/SourceEngineQueryCacher), which is ideal for resource-constrained environments like small VPS instances.

## Motivation

The primary motivation for this project was to create a simple, efficient solution for caching A2S queries for a game server running behind a home network and exposed to the internet via a VPS tunnel. Existing solutions were often resource-intensive, with the JVM-based `SourceEngineQueryCacher` consuming a significant amount of memory. `a2s-cache` is designed to be a lightweight and performant alternative, consuming minimal system resources.

## Usage

### Building the Project

To build the project, you will need to have [Go](https://go.dev/) installed. Once Go is installed, you can build the project using the following command:

```bash
go build -o a2s-cache ./cmd/a2s-cache
```

This will produce a binary named `a2s-cache` in the current directory.

### Running the Server

The server can be run as follows:

```bash
./a2s-cache [flags]
```

The following command-line flags are available:

| Flag       | Description                      | Default     |
|------------|----------------------------------|-------------|
| `debug`    | Enable debug logs                | `false`     |
| `gameIP`   | IP address of the game server    | `1.2.3.4`   |
| `gamePort` | Port of the game server          | `27015`     |
| `ip`       | IP address for the UDP server to listen on | `127.0.0.1` |
| `port`     | Port for the UDP server to listen on | `9000`      |

### Example

```bash
./a2s-cache --gameIP=192.168.1.100 --gamePort=27015 --ip=0.0.0.0 --port=9000
```

This will start the cache server, listening on all network interfaces on port `9000`, and forwarding queries to a game server at `192.168.1.100:27015`.

## Firewall Configuration

To use `a2s-cache`, you will need to configure your firewall to redirect A2S query packets to the cache server instead of the game server. The following example demonstrates how to do this using `nftables` on a modern Linux distribution.

**Note:** The following is an example configuration and may need to be adjusted to fit your specific needs.

### nftables

The following configuration can be added to your `nftables` ruleset, which is typically located at `/etc/nftables.conf`:

```nft
#!/usr/sbin/nft -f

flush ruleset

table inet filter {
    chain input {
        type filter hook input priority filter;

        # Allow traffic to the game server
        udp dport 27015 accept;
        udp dport 27005 accept;

        # Allow traffic to the A2S cache
        udp dport 9000 accept;

        policy accept;
    }

    chain forward {
        type filter hook forward priority filter;
        policy accept;
    }

    chain output {
        type filter hook output priority filter;
        policy accept;
    }
}

table ip nat {
    chain prerouting {
        type nat hook prerouting priority dstnat; policy accept;

        # Redirect A2S queries to the cache server
        udp dport 27015 @ih,0,40 0xffffffff54 redirect to :9000;
        udp dport 27015 @ih,0,40 0xffffffff55 redirect to :9000;

        # Forward other traffic to the game server
        udp dport 27015 dnat to 10.0.0.2:27015;
        udp dport 27005 dnat to 10.0.0.2:27005;
    }
}
```

### iptables

If you still wish to use iptables, you can refer to the instructions in the README for [SourceEngineQueryCacher](https://github.com/hyperxpro/SourceEngineQueryCacher),
as the firewall configuration should be identical (just make sure you use the same port).


## Contributing

Contributions are welcome! If you find a bug or have a feature request, please open an issue on the [GitHub repository](https://github.com/Augiro/a2s-cache).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
