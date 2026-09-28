# cache-refresh-demo-sidecar

A minimal demo of refreshing the cache in every pod through Kafka.

```
POST /sidecar/cache/refresh {"id":"abc"}
   │
   ▼
handler.go  ── sends {"id":"abc"} ──►  Kafka topic "cache-refresh"
                                            │
                        ┌───────────────────┴───────────────────┐
                        ▼                                       ▼
              kafka.go consumer (pod-1)               kafka.go consumer (pod-2)
                        │                                       │
                        ▼                                       ▼
              refresh.go → POST repo /cache/refresh   (up to 3 attempts, 2s apart)
```

| File | What it does |
|---|---|
| `main.go` | Reads settings, creates the pieces, starts the server and consumer, and handles shutdown |
| `handler.go` | `POST /sidecar/cache/refresh` → validates the body → sends it to Kafka |
| `kafka.go` | `Producer` (send) and `Consumer` (receive → call refresher) |
| `refresh.go` | Calls the repo refresh API, 3 attempts; if all fail, it logs |
| `tools/fakekafka` | In-memory Kafka on `localhost:9092` (no Docker) |
| `tools/fakerepo` | Fake repo refresh API on `localhost:8080` |

## Run the demo (4 terminals)

```bash
# 1. Kafka
go run ./tools/fakekafka

# 2. Fake repo (first 2 calls fail, to show retries)
FAIL_TIMES=2 go run ./tools/fakerepo

# 3. Sidecar "pod-1"
POD_NAME=pod-1 go run .

# 4. Sidecar "pod-2" (to show that every pod refreshes)
POD_NAME=pod-2 HTTP_ADDR=:8091 go run .
```

Send a refresh to **one** pod:

```bash
curl -X POST localhost:8090/sidecar/cache/refresh -d '{"id":"abc"}'
# {"id":"abc","status":"sent to kafka"}
```

**Both** pods log that they received the message and refreshed:

```
[api] sent id=abc to kafka                      (pod-1 only)
[consumer] received id=abc                      (pod-1 and pod-2)
[refresh] attempt 1/3 failed id=abc: repo returned status 500
[refresh] SUCCESS id=abc (attempt 2)
```

To show the "all attempts failed" case, restart the fake repo with `FAIL_TIMES=99`.

## Run with Docker (real Kafka + your Java app)

`docker-compose.yml` starts 3 containers: `kafka`, `pod1` and `pod2`. Each pod container runs **both** your
Java app and this sidecar (started by `start.sh`), so the sidecar calls the app at `localhost:8080`.

| | Java app | Sidecar |
|---|---|---|
| pod-1 | `localhost:8080` | `localhost:8090` |
| pod-2 | `localhost:8081` | `localhost:8091` |

**1. The Java app** needs `POST /cache/refresh`. The sidecar sends `{"id":"abc"}` as a JSON body:

```java
@PostMapping("/cache/refresh")
public ResponseEntity<Void> refresh(@RequestBody Map<String, String> body) {
    log.info("Cache refresh called, id={}", body.get("id"));
    // reload cache here
    return ResponseEntity.ok().build();   // any 2xx = success, anything else = sidecar retries
}
```

If your endpoint uses a different path, set `REFRESH_URL=http://localhost:8080/your/path`.

**2. Copy your jar in and start everything**

```bash
cp /path/to/your-java-app/target/your-app-0.0.1-SNAPSHOT.jar ./app.jar
docker compose up --build
```

Wait until the Java app has started and the sidecar has logged `[consumer] started` in both pods.

**3. Test A: call the API of either sidecar**

```bash
curl -X POST localhost:8090/sidecar/cache/refresh -d '{"id":"test-1"}'   # pod-1's sidecar
curl -X POST localhost:8091/sidecar/cache/refresh -d '{"id":"test-2"}'   # pod-2's sidecar
```

Each call should make **both** Java apps log `Cache refresh called`.

**4. Test B: publish directly to Kafka (no sidecar API)**

```bash
echo '{"id":"from-kafka"}' | docker exec -i kafka \
  /opt/kafka/bin/kafka-console-producer.sh --bootstrap-server kafka:9092 --topic cache-refresh
```

**Both** Java apps should again log the call.

**Watch the logs**

```bash
docker compose logs -f pod1 pod2    # Java and sidecar logs together; sidecar lines start with [api]/[consumer]/[refresh]
```

**Stop:** `docker compose down`

## Settings

| Env var | Default |
|---|---|
| `KAFKA_BROKERS` | `localhost:9092` (comma-separated) |
| `KAFKA_TOPIC` | `cache-refresh` |
| `REFRESH_URL` | `http://localhost:8080/cache/refresh` |
| `HTTP_ADDR` | `:8090` |
| `POD_NAME` | the machine's hostname |

## Not in the demo (see `../cache-refresh-sidecar` for these)

- API key auth, input limits
- failure topic, metrics, status endpoint
- jitter, backoff, slow retries, replacing an older pending event with a newer one
- a refresh in progress blocks the consumer (fine for a demo)
- Kafka TLS/SASL, Kubernetes manifests (in Kubernetes, the app and sidecar would be two containers in one pod, not one container)
