# Beam

Beam is a simple peer-to-peer file/folder transfer CLI tool written in Go. It uses UDP broadcast for peer discovery and TCP for reliable file streaming over a local network.

## Features

- Zero-configuration LAN discovery via UDP broadcast
- Reliable file transfer over TCP
- Rich user experience with real-time progress bar
- Support for both single file and entire folder transfers
- Smart handling of .gitignore patterns during folder transfers

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

Make sure your `$GOPATH/bin` (or Go's module bin directory) is in your `PATH`.

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
beam send -path="path/to/file/or/folder"
```

Steps:

1. **Discovery**: Scans the LAN and lists all active receivers:
   ```
   1. receiver-host -- 192.168.1.42:54321
   2. other-host    -- 192.168.1.43:54322
   ```
2. **Select**: Enter the ID of the target host (e.g., `1`).
3. **Transfer**: Streams the file over TCP. When complete, you'll see:
   ```
   ✅ Transfer complete: 1 file(s).
   ```

## Examples

Sending a 2 GB file

![Screen Recording Jun 14 2025](https://github.com/user-attachments/assets/c50af6e2-a3ad-4f96-8b97-172d0772080f)

Start a receiver on a fixed port:

```bash
beam receive
```

Send a file:

```bash
beam send -path="~/Downloads/picture.jpg"
```

