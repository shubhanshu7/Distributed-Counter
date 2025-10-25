# Distributed In-Memory Counter with Service Discovery (Go)

This project implements a **distributed in-memory counter** with **dynamic service discovery**, **eventual consistency** written entirely in Go.

---

##  Overview
- Nodes auto-discover each other via a simple `/join` + `/heartbeat` protocol.
- Each node maintains a  **G-Counter** to ensure eventual consistency.
- Increments propagate asynchronously to all peers with retries and exponential backoff.
- Handles network partitions and re-syncs via `/state` merges.

---

##  Architecture
### 1. Service Discovery
- **Join Protocol:** A new node calls `/join` on a known peer, gets the cluster list, and merges it.
- **Heartbeats:** Nodes periodically ping each other; unresponsive peers are pruned.
- **Peer Sync:** Peer lists converge dynamically as nodes join or leave.

### 2. Distributed Counter
- Implements a **G-Counter** (grow-only counter).
- Each node tracks its own component count; global = sum of all components.
- Increments are deduplicated via per-node sequence numbers.
- Propagation retries use exponential backoff.

### 3. Failure Handling
- Network partitions: nodes continue local ops; states reconcile via `/state` merge.
- Down peers: updates queue until the peer returns.

---

##  API Endpoints
| Endpoint | Method | Description |
|-----------|---------|--------------|
| `/join` | POST | Register node and return known peers |
| `/heartbeat` | POST | Health check ping |
| `/peers` | GET | List all active peers |
| `/increment` | POST | Increment counter and broadcast |
| `/count` | GET | Get local and global counter |
| `/apply` | POST | Receive propagated update |
| `/state` | GET | Return full CRDT map |

---

## 🧰 How to Run Locally
```bash
go build ./cmd/counterd

# Start node 1
./counterd --addr=:8081

# Start node 2
./counterd --addr=:8082 --peers=localhost:8081

# Start node 3
./counterd --addr=:8083 --peers=localhost:8081,localhost:8082
