# Xdebug (DBGp) proxy

This is Xdebug dbgp proxy written in Go. This command line tool helps multiuser debugging by routing debugging request to PHP IDEs as described in [here](https://www.jetbrains.com/help/phpstorm/multiuser-debugging-via-xdebug-proxies.html)

The original [DBGp Proxy Tool](https://xdebug.org/docs/dbgpProxy) can get the job done, also there is [Komodo Remote Debugging Package](https://code.activestate.com/komodo/remotedebugging/) which is serve the same purpose, i think.

So, this just another implementation with Go, aims for efficiency and performance, with my motivation of getting more experience using Go =)

**References:**
* [Xdebug documentation](https://xdebug.org/docs/dbgp#just-in-time-debugging-and-debugger-proxies)

## Installation

```console
$ go get -v github.com/frengky/dbgpxy/...
```

## Running

Run the proxy to listen for Xdebug on port `9003` and listen for IDE registration on port `9033`
```console
$ dbgpxy -d 0.0.0.0:9003 -r 0.0.0.0:9033
```
