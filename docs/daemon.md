# What is gshell daemon

gshell daemon should be running on every system to enable this system to join the service
network so that the system becomes a visible and accessable node by other gshell enabled
nodes.

The daemon and all the services running on the same node share one unique `Provider ID`,
currently it is the first physical network card's MAC address(may change in the future).
`Provider ID` is used to identify the node in the service network.

A service is defined at programing time as {"publisher", "service"}, but at runtime, there
can be many gshell enabled nodes providing the same service at the same time, so the new
tuple {"publisher", "service", "provider"} is then used to uniquely locate a service in the
network.

On a gshell enabled node, `gshell list` command will list all the visible services, for example
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

Or use the official github gshell binary:

```shell
$ mkdir -p gshell/bin
$ wget https://github.com/godevsig/gshellos/releases/latest/download/gshell.amd64 -O bin/gshell
$ chmod +x bin/gshell
```

See help first:

```shell
$ bin/gshell daemon -h
Usage of daemon [options]
        Start local gshell daemon:
  -bcast string
        broadcast port for LAN
  -invisible
        make gshell daemon invisible in gshell service network
  -registry string
        root registry address
  -repo string
        code repo local path or https address in format site/org/proj/branch
  -root
        enable root registry service
  -update string
        url of artifacts to update gshell, require -root
  -wd string
        set working directory (default "/var/tmp/gshell")
```

- use `-bcast` if you want to enable gshell discovery in LAN scope.
- use `-registry` if you want to enable gshell discovery in WAN scope.
  NOTE: in WAN scope, a root registry is required to run, see below.
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
bin/gshell -loglevel info daemon -wd rootregistry -registry Your_Server_IP:11985 -bcast 9923 -root -repo github.com/godevsig/ghub/master -update http://10.10.10.10:8088/gshell/release/latest/ &

# after gshell daemon started, check code repo contents
bin/gshell repo ls
```

- registry: the same ip where you run gshell root registry
- root: root registry mode
- repo addr: specify central repo address where .go files reside
- update addr: automatically update gshell binary from the address, which should contain:
  `gshell.386 gshell.amd64 gshell.arm64 gshell.mips64 gshell.ppc gshell.ppc64 md5sum rev`

## Example: deploy coordinated gshell daemons

Follow the same steps of the root registry, except `gshell daemon` command:

```shell
# in gshell work dir
cd /path/to/gshell

# run gshell daemon
bin/gshell -loglevel info daemon -registry 10.10.10.10:11985 -bcast 9923 &

# or better to test if gshell deamon is already running
bin/gshell info || bin/gshell -loglevel info daemon -registry 10.10.10.10:11985 -bcast 9923 &
```

- registry: specify the root registry IP address, enables scope WAN
- bcast: LAN broadcast port, enables scope LAN

## Example: deploy standalone gshell daemon

If you decide to deploy gshell daemon on your Linux PC or inside a VM or a docker container only
in standalone mode, use below commands:

```
# in gshell work dir
cd /path/to/gshell

# start daemon without either registry address or LAN broadcast port will
# put the daemon in scope Process and OS only
bin/gshell daemon &
```

## Example: deploy gshell daemon in scope LAN

Adding `-bcast port` on starting gshell daemon then makes this daemon and all the services under
it scope LAN visible to the other gshell systems that also started with the same broadcast port:

```
# in gshell work dir
cd /path/to/gshell

# start daemon also in scope LAN
bin/gshell daemon -bcast 9923 &
```

## Auto update gshell binary

After started, gshell daemon will always try to update itself automatically,
so after the first time gshell daemon is started, just leave it running, closing the terminal is ok,
gshell daemon will be still running in background.

To be able to get auto update working, gshell root regitstry should be started with `-update` option,
and a http file service should be running, this is done by starting a gshell app:

For example, to autoupdate gshell binary from official github repo, here we don't need to start our own http file server,
we just use github download page:

```
bin/gshell -loglevel info daemon -wd rootregistry -registry 10.10.10.10:11985 -bcast 9923 -root -repo github.com/godevsig/ghub/master -update https://github.com/godevsig/gshellos/releases/latest/download/ &
```

Another example is autoupdating from private http file server:

```shell
cd /path/to/gshell
# start root gshell daemon
bin/gshell -loglevel info daemon -wd rootregistry -registry 10.10.10.10:11985 -bcast 9923 -root -repo github.com/godevsig/ghub/master -update http://10.10.10.10:8088/gshell/release/latest/ &
# start file server on the same node(10.10.10.10) of root gshell daemon
bin/gshell run util/fileserver/cmd/fileserver.go -dir /path/contains/gshell/release
```

The file server should contain:

```
$ ls
gshell.386  gshell.aarch64  gshell.amd64  gshell.arm64  gshell.i386  gshell.mips64  gshell.ppc  gshell.ppc64  gshell.x86_64  md5sum  rev

$ cat md5sum
94530ecb0cc832039cb47011469038fc  bin/gshell.386
f8923dfb13a049c7747161e54768bddf  bin/gshell.aarch64
13209e10228da7c65d0c1bd93543624a  bin/gshell.amd64
f8923dfb13a049c7747161e54768bddf  bin/gshell.arm64
94530ecb0cc832039cb47011469038fc  bin/gshell.i386
e8e7693e741c3388d9ae437627c82eb4  bin/gshell.mips64
7f2645fe507859e1e3a35d603b3691d2  bin/gshell.ppc
895ed26b51b98c1b2ef2df32df6aa7b1  bin/gshell.ppc64
13209e10228da7c65d0c1bd93543624a  bin/gshell.x86_64

$ cat rev
957ca365d0ecd26846d15733203d3e3bfc4e9645
```

### Disable auto update

When debugging gshell itself, we don't want auto update working:

```gshell
GSHELL_NOUPDATE=1 bin/gshell -loglevel info daemon -wd .working -registry 10.10.10.10:11985 -bcast 9923 &
```

# Q&A

1. service not found: godevsig_gshellDaemon  
   Start gshell daemon first before any other commands.
1. Start gshell daemon failed with "listen unix @adaptiveservice/xxxxx: address already in use"  
   There is an old gshell daemon still running. `pkill -SIGINT gshell` to kill the old one, and then start the new one.
