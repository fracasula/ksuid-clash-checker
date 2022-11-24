# ksuid-clash-checker

Start server first:

```bash
cd cmd/server
go run main.go 8181 23
```

Start any number of clients from different computers too as long as connected to a network and visible:

```bash
cd cmd/client
go run main.go 192.168.1.34:8181 20
```

The above IP address must be the IP address of the server.
The server is the one checking for clashes.
