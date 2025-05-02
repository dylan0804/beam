# Beam

Beam is a simple peer-to-peer file transfer CLI tool written in Go. It uses UDP broadcast for peer discovery and TCP for reliable file streaming over a local network.

## Features

- Zero-configuration LAN discovery via UDP broadcast
- Reliable file transfer over TCP
- Cross-platform (Windows, macOS, Linux)
- Efficient streaming with configurable buffer sizes

## Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/dylan0804/beam.git
cd beam
go build -o beam ./cmd/beam
```

Or install directly via `go install`:

```bash
go install github.com/dylan0804/beam/cmd/beam@latest
```

Make sure your `$GOPATH/bin` (or Go’s module bin directory) is in your `PATH`.

## Usage

Beam supports two modes: **receive** (listen) and **send**.

### Receive Mode

Start a receiver to accept incoming file transfers:

```bash
beam receive
```

The receiver will broadcast its presence on UDP port `9999` every second, advertising its hostname and listening port.

### Send Mode

Send a file to a discovered receiver:

```bash
beam send -path="path/to/file"
```

Steps:

1. **Discovery**: Scans the LAN and lists all active receivers:
   ```
   1. receiver-host -- 192.168.1.42:54321
   2. other-host    -- 192.168.1.43:54322
   ```
2. **Select**: Enter the ID of the target host (e.g., `1`).
3. **Transfer**: Streams the file over TCP. When complete, you’ll see:
   ```
   ✅ Transfer complete: 12.3 KB received to path/to/download.
   ```

## Configuration

- **Broadcast port**: UDP `9999` (hardcoded)
- **Buffer size**: 32 KiB (adjustable in code)

## Examples

Start a receiver on a fixed port:

```bash
beam receive
```

Send a file:

```bash
beam send -path="~/Downloads/picture.jpg"
```

