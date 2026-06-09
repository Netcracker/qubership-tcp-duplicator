# TCP duplicator

`tcp-duplicator` is a TCP proxy, so it does not care which higher-level protocol you are using.
It mirrors the data to all configured servers.

`tcp-duplicator` only supports uncompressed and non-chunked payloads.
Each message needs to be delimited with a null byte (\0) when sent in the same TCP connection.

## Prerequisites

The installation prerequisites are as follows.

### HWE Requirements

Open port:

* `listen_port` - for logs inputs (by default: `12202`)

### HWE and Limits

`tcp-duplicator` is installed on the VM as a docker container.

It requires:

* CPU - `200 millicores`
* RAM - `~150-200 MB` (with default parameters)

### Software Requirements

* `docker`

## Configuration

The `tcp-duplicator` reads parameters from ENV. It supports the following ENVs:

<!-- markdownlint-disable line-length -->
| Name              | Description                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------ |
| TCP_ADDRESSES     | List of TCP addresses for logs outputs (comma-separated), for example: `10.10.10.10:12201,10.10.10.11:12201` |
| LISTEN_PORT       | Port for logs inputs                                                                                         |
| FLUSH_INTERVAL    | The interval between data flushes                                                                            |
| BUFFER_LIMIT_SIZE | The length limit of the buffer                                                                               |
| RETRY_COUNT       | The number of retries for flushes                                                                            |
<!-- markdownlint-enable line-length -->
