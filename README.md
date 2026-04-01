# Peer-to-Peer File Sharing

The project is still in-progress.

## Overview

This MVP includes two primary protocols:

- **Transfer Protocol (`/p2pfs/get/1.0.0`)**: Stream-based protocol handling file downloads.

- **Index Protocol (`/p2pfs/index/1.0.0`)**: Stream-based request protocol allowing peers to manually verify what files a target peer is serving.

It uses **GossipSub** (`p2pfs/announce/1.0.0`) to run periodic broadcasts announcing local files. Each file is identified by a CID derived from its raw bytes, while filenames are kept as metadata for display and local saves. Peers ingest these announcements to maintain an ephemeral `providers` map in-memory.

## Getting Started

Install dependencies and compile:

```bash
go mod tidy
go build -o p2pfs
```

The `p2pfs` binary supports two usage patterns: running as a background daemon, or issuing stateless CLI client commands to a running daemon. Because the CLI tool works by talking to the background daemon on the same machine, the daemon exposes a local UNIX RPC socket.

### Start a Daemon

A node can share files in its `export-dir`.

```bash
./p2pfs daemon -listen /ip4/127.0.0.1/tcp/4001 -export_dir ./my_files
```

### Bootstrapping

To bootstrap a new daemon, pass a comma-separated list of known `/ip4/.../p2p/<PeerID>` multiaddresses to the `-bootstrap` flag.

```bash
./p2pfs daemon -listen /ip4/127.0.0.1/tcp/4002 -export_dir ./my_files -bootstrap <P2P_MULTIADDR_FROM_SEED>
```

### CLI Commands

Once the daemon is up and a network has been established over GossipSub, query and fetch using the CLI:

- `whohas`: Ask local daemon's provider index who has a specific CID.
```bash
./p2pfs whohas <CID>
```

- `fetch`: Tell daemon to download content by CID from the network into its local `export_dir`.

```bash
./p2pfs fetch <CID>
```

- `list`: Connect to a remote peer explicitly and use the Index protocol to verify what they are serving, including filename, CID, and size.

```bash
./p2pfs list --peer <REMOTE_MULTIADDR>
```

## MVP Demo

To run the demo (which spins up Peer A, Peer B, and Peer C, and generates a test `foo.txt` file):
```bash
./demo.sh start
```

Watch the terminal as it starts the daemons. Once it completes, first list a remote peer to see the CID for `foo.txt`, then use that CID from Peer C.

```bash
./p2pfs list   --rpc /tmp/p2pfsC.sock --peer <REMOTE_MULTIADDR>
./p2pfs whohas --rpc /tmp/p2pfsC.sock <CID>
./p2pfs fetch  --rpc /tmp/p2pfsC.sock <CID>
cat peerC_export/foo.txt
```

To clean up the spawned log files, temp socket files, `export` directories, and abruptly kill all `p2pfs` processes:

```bash
./demo.sh clean
```
