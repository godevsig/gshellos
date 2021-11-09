# What is gshell daemon

gshell daemon should be running on every system to enable this system to join the service
network so that the system is visible and accessable by other gshell enabled systems.

The daemon and all the services running on the same system share one unique `Provider ID`,
currently it is the first physical network card's MAC address(may change in the future).
`Provider ID` is used to identify the system in the service network.

A service is defined at programing time as {"publisher", "service"}, but at runtime, there
can be many gshell enabled systems providing the same service at the same time, so the new
tuple {"publisher", "service", "provider"} is then used to uniquely locate a service in the
network.

On a gshell enabled system, `gshell list` command will list all the visible services, for example
below output:

```
$ gshell list
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
builtin                   IPObserver                self          1000
builtin                   LANRegistry               self            11
builtin                   providerInfo              self            11
builtin                   registryInfo              self            11
builtin                   reverseProxy              fa163ecfb434   100
builtin                   reverseProxy              self          1100
builtin                   serviceLister             self            11
godevsig                  codeRepo                  self          1111
godevsig                  grg-owcaxi.v1.1.3         self            10
godevsig                  gshellDaemon              00198f936ea2  1000
godevsig                  gshellDaemon              00e3df230009  1000
godevsig                  gshellDaemon              184a6fefbbba  1000
godevsig                  gshellDaemon              fa163ecfb434  1100
godevsig                  gshellDaemon              self          1111
godevsig                  updater                   self          1111
```

- `builtin` publisher is actually [adaptiveservice](https://github.com/godevsig/adaptiveservice)
- `godevisg` publisher is actually [gshellos](https://github.com/godevsig/gshellos)
- gshell daemon itself is also a service, there are 5 systems(providers) providing "gshellDaemon"
  service according to the above output.
- `WLOP` stands for `WAN LAN OS PROCESS` scopes, showing in which scope the service is available.

# Deploy gshell deamon

`make full` or `make lite` to build gshell binary.
See [here](docs/interpreter.md) for build types details.

Copy the gshell binary to target system, the first thing we should do is starting gshell daemon:

```shell
$ mkdir -p gshell/bin
# copy gshell binary to gshell/bin
```

See help first:

```shell
$ gshell daemon -h
Usage of daemon [options]
        Start local gshell daemon:
  -bcast string
        broadcast port for LAN
  -registry string
        root registry address
  -repo string
        code repo https address in format site/org/proj/branch, require -root
  -root
        enable root registry service
  -update string
        url of artifacts to update gshell, require -root
```

- use `-bcast` if you want to enable gshell discovery in LAN scope
- use `-registry` if you want to enable gshell discovery in WAN scope
  NOTE: in WAN scope, a root registry is required to run, see below
- `-root` makes this gshell daemon root registry, only one "public" system can be root.
  Public means all the other gshell enabled systems can have IP connectivity to the root.

## Start gshell daemon under unprivileged user

If you run gshell daemon as normal user, you are exposing your user permissions to all gshell clients.
The current quick solution is run gshell daemon under nobody:nogroup permission, which is done
by below commands:

```
cd gshell
sudo chown -R nobody:nogroup . && sudo chmod ugo+ws bin/gshell
```

## Example: deploy root registry for WAN

```shell
# in gshell work dir
cd /path/to/gshell

# kill old gshell daemon if necessary
pkill -SIGINT gshell

# setuid to nobody
su root
chown -R nobody:nogroup . && chmod ugo+s bin/gshell
exit

# start gshell daemon at log level info
bin/gshell -wd rootregistry -loglevel info daemon -registry 10.10.10.10:11985 -bcast 9923 -root -repo github.com/godevsig/grepo/master -update http://10.10.10.10:8088/gshell/release/latest/%s &
```

- registry: the same ip where you run gshell root registry
- root: root registry mode
- repo addr: specify central repo address where .go files reside
- update addr: automatically update gshell binary from the address, which should contain:
  `gshell.386 gshell.amd64 gshell.arm64 gshell.mips64 gshell.ppc gshell.ppc64 md5sum rev`

## Example: deploy coordinated daemons

## Auto update gshell binary

After started, gshell daemon will always try to update itself automatically,
so after the first time gshell daemon is started, just leave it running, closing the terminal is ok,
gshell daemon will be still running in background.

To be able to get auto update working, gshell root regitstry should be started with `-update` option,
and a http file service should be running, this is done by starting a gshell app:

```shell
cd /path/to/gshell
bin/gshell run app-http/fileserver/fileserver.go -dir /path/to/gshell/release
```

### Disable auto update

When debugging gshell itself, we don't want auto update working:

```gshell
GSHELL_NOUPDATE=1 bin/gshell -loglevel info -wd .working daemon -registry 10.10.10.10:11985 -bcast 9923 &
```

# Q&A

1. service not found: godevsig_gshellDaemon
   Start gshell daemon first before any other commands.
1. Start gshell daemon failed with "socket already exists: [/var/tmp/adaptiveservice/builtin_serviceLister.sock]"
   There is an old gshell daemon still running. `pkill -SIGINT gshell` to kill the old one, and then start the new one.
1. gshell still failed to start?
   Try `rm -rf /var/tmp/adaptiveservice` and then restart gshell daemon
