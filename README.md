# msqp

[`msqp`][pkg-msqp] is a Go implementation of Valve's legacy
[**Master Server Query Protocol**][valve-msqp] (MSQP),
UDP protocol used by older Steam game browsers to retrieve server addresses.
The module provides a client, packet codec,
filter builder, and a small server implementation.

> [!WARNING]
> MSQP is a historical protocol, not a current Steam server-discovery API.  

For current Steam server discovery,
see the [Steam Web API server-list client][steamserverlist]
or the [Steam Connection Manager (CM) client][steamcm].

## When to use it

Use this module when maintaining an old game
or compatible client that still speaks MSQP,
or when implementing a private/community master server for such a client.
A game must support configuring or redirecting its master-server endpoint;
this package does not change a game's configuration
or replace Steam's infrastructure.

For current Steamworks server browsing, see the official
[ISteamMatchmakingServers documentation][matchmaking-servers].
That API and MSQP are different interfaces.

## Installation

Add the dependency with:

```sh
go get github.com/woozymasta/msqp@latest
```

## Use

Query a reachable MSQP master.
Replace the example endpoint with one you operate or have verified;
the placeholder is not a public service.

```go
func run() error {
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()

  client, err := msqp.NewClient("127.0.0.1:27011")
  if err != nil {
    return err
  }
  defer client.Close()

  request := msqp.Request{
    Region: msqp.Region(0xff), // all regions
    Filter: `\appid\730\secure\1`,
  }
  for result := range client.Stream(ctx, request) {
    if result.Err != nil {
      return result.Err
    }
    for _, address := range result.Page.Servers {
      fmt.Println(address)
    }
  }
  return nil
}
```

The `msqp/server` package can serve an address list or a custom handler:

```go
func serve() error {
  addresses := []netip.AddrPort{
    netip.MustParseAddrPort("192.0.2.10:27015"),
  }
  handler := server.NewStaticHandler(addresses)
  master, err := server.NewServer(handler)
  if err != nil {
    return err
  }

  conn, err := net.ListenPacket("udp4", ":27011")
  if err != nil {
    return err
  }
  defer conn.Close()
  return master.Serve(conn)
}
```

## Protocol details

* `Client.Query` fetches one page.
  `Client.Stream` follows continuation cursors until completion or error;
  receive from the channel until it closes,
  or cancel the context if stopping early.
* MSQP carries IPv4 server addresses over UDP.
  It does not provide encryption, authentication, or reliable delivery.
* `Region` is the raw one-byte legacy region code; `0xff` requests all regions.
  `Filter` can be built with `msqp.Filter` or supplied as a raw expression.
* The server applies request/response size limits and caps concurrent handlers.

> [!CAUTION]
> The UDP server has no built-in authentication or rate limiting.  
> Restrict who can reach it with network controls
> before exposing it to untrusted clients.

## Documentation and tests

* [Valve Developer Community: Master Server Query Protocol][valve-msqp]
* [Steamworks: ISteamMatchmakingServers][matchmaking-servers]
* [`msqp` on pkg.go.dev][pkg-msqp]
* [`msqp/server` on pkg.go.dev][pkg-msqp-server]

Run local tests with `go test ./...`.
The opt-in `msqp_live` tests send UDP queries
to uncommented addresses in [`testdata/masters.txt`][master-list]:

```sh
go test -tags msqp_live ./... -count=1
```

> [!NOTE]
> [`testdata/masters.txt`][master-list]
> keeps historical Valve endpoints for reference
> and a separately marked community endpoint for opt-in testing.
> The historical Valve entries are commented out
> and are not expected to be live.  
> Community endpoint availability can change.

<!-- links -->

[pkg-msqp]: https://pkg.go.dev/github.com/woozymasta/msqp
[pkg-msqp-server]: https://pkg.go.dev/github.com/woozymasta/msqp/server
[valve-msqp]: https://developer.valvesoftware.com/wiki/Master_Server_Query_Protocol
[matchmaking-servers]: https://partner.steamgames.com/doc/api/ISteamMatchmakingServers
[master-list]: testdata/masters.txt
[steamserverlist]: https://github.com/woozymasta/steamserverlist
[steamcm]: https://github.com/woozymasta/steamcm
