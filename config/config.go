package config
// //All the constants used to be housed here for easy change of system
const (
	NUMBER_OF_VNODES = 4;
	MAX_KEY = 100;
    REPLICATION_FACTOR = 3;
	RW_FACTOR = 1;
	RING_URL = "localhost:5001"
	PING_INTERVAL = 1
	TIMEOUT_INTERVAL = 1
	FAILURE_THRESHOLD = 20
	ADD_NODE_URL = "add-node"
	NEW_RING_ENDPOINT = "new-ring"
	RING_REGISTER_ENDPOINT = "set-ring"
	STETHO_URL = "http://localhost:5000"
	STETHO_SERVER_PORT = "5000"
	RING_SERVER_PORT = "5001"
	RING_MAX_ID = 64
	NODESERVER_REGISTER_ENDPOINT = "add-node"
	NODESERVER_WARMUP_DURATION = 3
	FAINT_NODE_ENDPOINT = "faint-node"
	REMOVE_NODE_ENDPOINT = "remove-node"
	REVIVE_NODE_ENDPOINT = "revive-node"
)
// type Constants struct{
// 	RING_URL string
// 	RING_SERVER_PORT string
// 	NUMBER_OF_VNODES int
// 	MAX_KEY int
// 	REPLICATION_FACTOR int
// 	RW_FACTOR int
// 	REGISTER_ENDPOINT string
// 	STETHO_URL string
// 	STETHO_SERVER_PORT string 
// 	RING_SERVER_PORT string
// 	PING_INTERVAL int
// 	TIMEOUT_INTERVAL int
// 	FAILURE_THRESHOLD int
// 	ADD_NODE_URL string
// 	NEW_RING_ENDPOINT string
// }

// func main(){
// 	consts = Constants{}
// 	consts.NUMBER_OF_VNODES = 4;
// 	consts.MAX_KEY = 100;
//     consts.REPLICATION_FACTOR = 3;
// 	consts.RW_FACTOR = 1;
// 	consts.RING_URL = "localhost:5001"
// 	consts.PING_INTERVAL = 1
// 	consts.TIMEOUT_INTERVAL = 1
// 	consts.FAILURE_THRESHOLD = 20
// 	consts.ADD_NODE_URL = "add-node"
// 	consts.RING_SERVER_PORT = "5001"
// 	consts.NEW_RING_ENDPOINT = "new-ring"
// 	consts.REGISTER_ENDPOINT = "set-ring"
// 	consts.STETHO_URL = "http://localhost:5000"
// 	consts.STETHO_SERVER_PORT = "5000"
// 	consts.RING_SERVER_PORT = "5001"
// }