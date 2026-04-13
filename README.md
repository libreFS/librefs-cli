# lc — libreFS Client

[![license](https://img.shields.io/badge/license-AGPL%20V3-blue)](https://github.com/libreFS/mc/blob/master/LICENSE)

Forked from `minio/mc` — the libreFS CLI client, rebranded as `lc`.

`lc` provides a modern alternative to UNIX commands like `ls`, `cat`, `cp`, `mirror`, `diff`, `find` etc. It supports filesystems and Amazon S3-compatible cloud storage (AWS Signature v2 and v4), and works seamlessly with [libreFS](https://github.com/libreFS/libreFS).

```
  alias      manage server credentials in configuration file
  admin      manage libreFS servers
  anonymous  manage anonymous access to buckets and objects
  batch      manage batch jobs
  cp         copy objects
  cat        display object contents
  diff       list differences in object name, size, and date between two buckets
  du         summarize disk usage recursively
  encrypt    manage bucket encryption config
  event      manage object notifications
  find       search for objects
  get        get s3 object to local
  head       display first 'n' lines of an object
  ilm        manage bucket lifecycle
  idp        manage libreFS IDentity Provider server configuration
  license    license related commands
  legalhold  manage legal hold for object(s)
  ls         list buckets and objects
  mb         make a bucket
  mv         move objects
  mirror     synchronize object(s) to a remote site
  od         measure single stream upload and download
  ping       perform liveness check
  pipe       stream STDIN to an object
  put        upload an object to a bucket
  quota      manage bucket quota
  rm         remove object(s)
  retention  set retention for object(s)
  rb         remove a bucket
  replicate  configure server side bucket replication
  ready      checks if the cluster is ready or not
  sql        run sql queries on objects
  stat       show object metadata
  support    support related commands
  share      generate URL for temporary access to an object
  tree       list buckets and objects in a tree format
  tag        manage tags for bucket and object(s)
  undo       undo PUT/DELETE operations
  update     update lc to latest release
  version    manage bucket versioning
  watch      listen for object notification events
```

## Docker Container

```
docker pull ghcr.io/libreFS/mc
docker run ghcr.io/libreFS/mc ls play
```

## Install

Download the latest binary from the [releases page](https://github.com/libreFS/mc/releases).

```bash
# Linux (amd64)
curl -Lo lc https://github.com/libreFS/mc/releases/latest/download/linux-amd64/lc
chmod +x lc
sudo mv lc /usr/local/bin/

# macOS (arm64)
curl -Lo lc https://github.com/libreFS/mc/releases/latest/download/darwin-arm64/lc
chmod +x lc
sudo mv lc /usr/local/bin/
```

## Quick Start

```bash
# Point lc at your libreFS server
lc alias set local http://localhost:9000 minioadmin minioadmin

# List buckets
lc ls local

# Copy a file
lc cp myfile.txt local/mybucket/
```

## Compatibility

`lc` is fully compatible with any S3-compatible server. All existing `mc` commands and flags work identically. The `MC_*` environment variables are preserved for backwards compatibility.

## Build from Source

```bash
go build -o lc .
```

## License

GNU AGPLv3 — see [LICENSE](LICENSE).
