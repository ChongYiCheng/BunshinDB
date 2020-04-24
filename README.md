# 50.041 DistSysProject BunshinDB
 
### Run Instructions 

```cassandraql
#To run a nodeServer
go run cmd/nodeServer.go <portNumber> <pathToDbFiles> <nodeId> 

#To run a client 
go run cmd/client.go <portNumber> 

```

#### Example Usage: 
To run a nodeServer
```cassandraq#
go run cmd/nodeServer.go 8080 /tmp/badger8080 1
help 
```
To run a client 
```cassandraql
go run cmd/client.go 9000
help 
```

Fault Detection 
```cassandraql
go run cmd/ringServer.go
go run cmd/stethoServer.go
go run cmd/nodeServerSample.go 

```