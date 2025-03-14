# queueinator

A command line tool to easily set TCP server queues that execute a given task remotely.

## Installation

```bash
make
sudo make install
```

Note that this compilation depends on go. The current development version can be seen in `go.mod`.

## Usage

* Server mode:

```bash
queueinator serve COMMAND PORT [-t CHECK_PERIOD] [-n NPROCS] [-b BUFFER_SIZE] [-e EXPIRE_TIME]
```

The server checks for new jobs once every `CHECK_PERIOD` seconds (def. 1.0).
Up to `NPROCS` processes are ran in parallel (def. 1).
Incoming data must be constrained to `BUFFER_SIZE` Mb (def. 10).
The job expires and is deleted if the client does not fetch the resulting data after `EXPIRE_TIME` seconds (def. 3600.0).

* Cleanup mode:

```bash
queueinator cleanup
```

In a server, removes all previous job folders.

* Client mode:

```bash
queueinator run IP PORT [-t CHECK_PERIOD] [-b BUFFER_SIZE]
```

Sends contents of current folder to server at `IP:PORT` with data limit of `BUFFER_SIZE` Mb.
Checks for process conclusion/fetches results once every `CHECK_PERIOD` seconds.
Exits when the process has been executed at the server.


