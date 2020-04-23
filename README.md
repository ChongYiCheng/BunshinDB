# 50.041 DistSysProject BunshinDB
 
### Run Instructions 

To run the nodeServer: 

```cassandraql
#To run a nodeServer
go run cmd/nodeServer.go <portNumber> <pathToDbFiles> <nodeId> 

#To run a client 
go run cmd/client.go <portNumber> 

```

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
