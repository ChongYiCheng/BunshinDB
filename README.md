# 50.041 DistSysProject BunshinDB
 
### Run Instructions 

```cassandraql
#To run a nodeServer
go run cmd/nodeServer.go <portNumber> <pathToDbFiles> <nodeId> <shouldRegister true|false> 

#To run a client 
go run cmd/client.go <portNumber> 

```

Note: The last argument, `shouldRegister` is either `"true"` or `"false"`. 
Set it to false if we want to simulate a revival of the node.  

#### Example Usage: 
To run a nodeServer
```cassandraq#
go run cmd/nodeServer.go 8081 /tmp/badger8081 1
help 
```
To run a client 
```cassandraql
go run cmd/client.go 8080 
help 
```

Fault Detection 
```cassandraql
go run cmd/ringServer.go
go run cmd/stethoServer.go
go run cmd/nodeServer 8081 /tmp/badger8081 1 true 

```