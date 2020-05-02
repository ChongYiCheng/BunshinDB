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
