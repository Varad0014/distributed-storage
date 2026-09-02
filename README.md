# Distributed Storage Service

A distributed file-storage API written in Go. The API stores file metadata in PostgreSQL and streams object bytes to three independently persisted storage-node containers. Storage nodes communicate with the API over authenticated HTTP, while a background repair worker uses PostgreSQL checksums as the authoritative source for detecting and repairing missing or corrupted replicas.

## Architecture

```text
Client
  |
  v
Main API (:8080)
  |-- PostgreSQL (:5432)
  |     `-- file ID, name, size, SHA-256 checksum, created time
  |
  `-- ReplicatedStorage
        |-- HTTP --> storage_node_1:8081 --> file_storage_1
        |-- HTTP --> storage_node_2:8081 --> file_storage_2
        `-- HTTP --> storage_node_3:8081 --> file_storage_3
```

Docker publishes the storage nodes on host ports `8081`, `8082`, and `8083` for local testing. Inside the Compose network, every storage node listens on port `8081`.

## Main features

- Streaming upload and download using Go's `io.Reader` and `io.Copy`
- PostgreSQL metadata separated from object storage
- Full replication across three storage nodes
- Read fallback when a storage node is unavailable
- SHA-256 integrity checks
- Metadata-authoritative repair of missing and corrupted replicas
- Periodic background synchronization and a manual synchronization endpoint
- Bearer-token authentication between the API and storage nodes
- Canonical UUID validation before filesystem access
- Upload-size limits, graceful API shutdown, health reporting, and HTTP timeouts
- Docker Compose environment with separate persistent volumes

## Requirements

- Go matching the version in `go.mod`
- Docker with Docker Compose
- `curl`

## Run with Docker Compose

Choose a storage-node token. The same token is provided to the API and every storage node.

```bash
export STORAGE_NODE_TOKEN='replace-this-with-a-long-random-secret'
docker compose up --build -d
```

Check container status:

```bash
docker compose ps
```

Follow logs:

```bash
docker compose logs -f app storage_node_1 storage_node_2 storage_node_3
```

Stop the stack without deleting stored data:

```bash
docker compose down
```

To also delete PostgreSQL and storage-node volumes:

```bash
docker compose down -v
```

The `-v` operation permanently deletes the local project data stored in Docker volumes.

## Test storage-node APIs directly

Storage-node endpoints require the bearer token. Requests without it should be rejected:

```bash
curl -i http://localhost:8081/health
```

Expected status:

```text
HTTP/1.1 401 Unauthorized
```

With authentication:

```bash
curl -i \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  http://localhost:8081/health
```

Expected response:

```text
Storage node is healthy
```

Use a canonical UUID when calling object endpoints directly:

```bash
OBJECT_ID='550e8400-e29b-41d4-a716-446655440000'
```

Upload an object to node 1:

```bash
curl -i \
  -X POST \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  --data-binary 'direct storage-node test' \
  "http://localhost:8081/objects/$OBJECT_ID"
```

List objects on node 1:

```bash
curl \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  http://localhost:8081/objects
```

Check existence without downloading the body:

```bash
curl -I \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects/$OBJECT_ID"
```

Calculate the stored checksum:

```bash
curl \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects-checksum/$OBJECT_ID"
```

Download the object:

```bash
curl \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects/$OBJECT_ID"
```

Delete the direct test object:

```bash
curl -i \
  -X DELETE \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects/$OBJECT_ID"
```

Nodes 2 and 3 are available from the host on `localhost:8082` and `localhost:8083`. Their internal container port remains `8081`.

## Test the main API

Create a local test file:

```bash
printf 'hello from distributed storage\n' > test-upload.txt
```

Upload it through the main API:

```bash
curl -sS \
  -F 'file=@test-upload.txt' \
  http://localhost:8080/files | tee /tmp/distributed-storage-upload.json
```

The response contains metadata similar to:

```json
{
  "id": "c72b7e43-923d-4a9d-936a-8ec02a55d3d7",
  "name": "test-upload.txt",
  "size": 31,
  "checksum": "...",
  "created_at": "..."
}
```

If `jq` is installed, capture the generated ID:

```bash
OBJECT_ID=$(jq -r '.id' /tmp/distributed-storage-upload.json)
echo "$OBJECT_ID"
```

Otherwise, copy the `id` from the upload response and set it manually:

```bash
OBJECT_ID='paste-the-returned-uuid-here'
```

List file metadata:

```bash
curl -sS http://localhost:8080/files
```

Download through the main API:

```bash
curl -sS \
  -o downloaded-test-upload.txt \
  "http://localhost:8080/files/$OBJECT_ID"
```

Compare the files:

```bash
cmp test-upload.txt downloaded-test-upload.txt
```

No output from `cmp` means the files are identical.

Check storage-node health through the main API:

```bash
curl -sS http://localhost:8080/health/storage
```

Delete through the main API:

```bash
curl -i \
  -X DELETE \
  "http://localhost:8080/files/$OBJECT_ID"
```

## Test replication and repair

Upload a fresh file through the main API and capture its `OBJECT_ID` as shown above. Confirm that all nodes contain it:

```bash
for port in 8081 8082 8083; do
  curl -I \
    -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
    "http://localhost:${port}/objects/$OBJECT_ID"
done
```

### Repair a missing replica

Delete only node 1's copy by calling the storage node directly. PostgreSQL metadata remains present:

```bash
curl -i \
  -X DELETE \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects/$OBJECT_ID"
```

Confirm node 1 reports `404`:

```bash
curl -I \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects/$OBJECT_ID"
```

Trigger repair immediately instead of waiting for the background worker:

```bash
curl -i -X POST http://localhost:8080/storage/sync
```

Confirm node 1 contains the repaired object:

```bash
curl -I \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects/$OBJECT_ID"
```

### Repair a corrupted replica

Record the correct checksum:

```bash
curl \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8082/objects-checksum/$OBJECT_ID"
```

Overwrite only node 1 with different bytes:

```bash
curl -i \
  -X POST \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  --data-binary 'corrupted replica' \
  "http://localhost:8081/objects/$OBJECT_ID"
```

Node 1's checksum should now differ. Trigger repair:

```bash
curl -i -X POST http://localhost:8080/storage/sync
```

Check node 1 again:

```bash
curl \
  -H "Authorization: Bearer $STORAGE_NODE_TOKEN" \
  "http://localhost:8081/objects-checksum/$OBJECT_ID"
```

Its checksum should once again match PostgreSQL's authoritative checksum and the healthy replicas.

## API summary

### Main API

| Method | Path | Description |
|---|---|---|
| `POST` | `/files` | Upload multipart field named `file` |
| `GET` | `/files` | List file metadata |
| `GET` | `/files/{id}` | Download a file |
| `DELETE` | `/files/{id}` | Delete a file and its replicas |
| `GET` | `/health/storage` | Show storage-node health |
| `POST` | `/storage/sync` | Trigger replica repair |

### Storage-node API

Every storage-node request requires `Authorization: Bearer <token>`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Node health check |
| `GET` | `/objects` | List object IDs |
| `POST` | `/objects/{id}` | Store or replace an object |
| `GET` | `/objects/{id}` | Download an object |
| `HEAD` | `/objects/{id}` | Check whether an object exists |
| `DELETE` | `/objects/{id}` | Delete an object |
| `GET` | `/objects-checksum/{id}` | Calculate an object's SHA-256 checksum |

## Automated tests

Run all unit tests:

```bash
go test ./...
```

Run with verbose output:

```bash
go test -v ./...
```

Run the race detector:

```bash
go test -race ./...
```

Run static analysis and formatting checks:

```bash
go vet ./...
gofmt -d -- *.go cmd internal
git diff --check
```

The storage tests use temporary directories and in-memory fake nodes. They do not require Docker or PostgreSQL.

Current unit coverage includes:

- Local storage save, open, checksum, list, existence, and delete lifecycle
- Rejection of unsafe object IDs
- Storage-node bearer-token middleware
- Read fallback when the first replica is unavailable
- Repair of missing and corrupt replicas from an authoritative checksum
- Safe failure when no replica matches the authoritative checksum

## Design decisions and tradeoffs

- **Storage abstraction:** `StorageNode` allows local, remote, and replicated implementations to be used through one interface.
- **Streaming:** file bytes move through `io.Reader` instead of being fully loaded into memory.
- **Temporary upload copy:** replication needs a reusable source because an `io.Reader` is normally consumed once.
- **Metadata authority:** PostgreSQL determines which objects should exist and which checksum is correct during repair.
- **Write policy:** uploads currently require every configured node to succeed. This favors replica consistency but reduces write availability.
- **Read policy:** reads try nodes sequentially until one responds successfully.
- **Internal authentication:** a shared bearer token protects storage-node endpoints. Production deployment should additionally use a private network and TLS.

## Known limitations and future scope

- Write objects through a temporary file and atomically rename them to prevent partial files from becoming visible.
- Verify a replica against PostgreSQL's expected checksum before serving a download.
- Add configurable read and write quorums so uploads can succeed while a minority of nodes are unavailable.
- Propagate request contexts through the entire storage interface for cancellation and per-operation deadlines.
- Replace the single PostgreSQL connection with `pgxpool` and use a migration tool for schema changes.
- Add a distributed lease or lock so multiple API instances do not run conflicting repair jobs.
- Cache checksums or maintain object manifests to avoid hashing full files during every synchronization pass.
- Add pagination, byte-range downloads, rate limiting, quotas, metrics, tracing, and structured logging.
- Deploy storage nodes on independent machines or availability zones; Docker Compose currently simulates distribution on one host.


