# Foundation contracts and migration

This release changes unsafe contracts deliberately. Upgrade gokitbox and its application
adapters together; do not mix old/new ID generators or RPC authentication in a rolling rollout.

## IDs

`uSnowflake.New` owns an exclusive node lease and persistent, disjoint timestamp windows.
The Redis allocator shares a project-wide pool across all roles, retains high-water marks,
fails on capacity/ownership/coordinator errors, and never uses modulo/random fallback.
Renewal failure is surfaced by `Wait`; gokit supervises it and shuts down the application.
Clock regression stops ID generation. A normal close releases ownership but retains the
high-water mark, so a replacement must advance beyond the old reserved window.

Before FIRST deployment, stop every old generator in the same uniqueness domain, ensure
its database/Redis state is understood, then explicitly create the new pool from gokit:

```sh
go run ./cmd/gokit-cli --config configs/app.yaml snowflake:init --empty-domain
```

This operation does not reset an existing pool. `--empty-domain` is an operator assertion,
not a check that a crashed/partitioned old deployment is gone. Never use initialization to
recover a lost allocator while old generators may still run. Redis must preserve committed
allocator state; arbitrary snapshot rollback/failover data loss is outside the guarantee.
Use a durable, non-evicting, suitably consistent coordinator and a controlled restore plan.
Project names define the uniqueness domain; projects sharing the same ID namespace must
share that domain. There are 1024 concurrent nodes. All time windows and IDs have bounds.

`Initialize`/`NextInt64`/`NextString` return errors; legacy `InitSnowflake`/`NewId*` fail closed
via panic because their signatures cannot report errors. Bare `MakeNode` is retired.

Business numbers use 128 CSPRNG bits in 26 uppercase Base32 characters plus a two-character
business prefix: default length28. `uNo.NewNumber` returns errors; old string helpers panic
on invalid prefix/length or entropy failure. No user-ID suffix is a uniqueness source.
Length is28..128, custom prefix <=32 ASCII alphanumeric/-/_. Keep DB unique constraints;
randomness provides probabilistic uniqueness, not a mathematical absence of collisions.
Validate channel-specific formats in the application adapter; historical numbers remain readable.

## Redis locks

```go
lease, err := redisdao.Acquire(ctx, client, "lock:order:123", 30*time.Second)
if err != nil { return err }
// Work must respect ownership loss. Use a fresh, bounded cleanup context.
cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
defer cancel()
return lease.Release(cleanup)
```

`Renew` and `Release` compare the random owner token atomically. Positive millisecond TTLs
are required. Old `TryLock`/ownerless `Unlock` return migration errors rather than silently
performing unsafe operations. A lock cannot undo an expired owner's external side effect;
use idempotency/fencing where required. The caller owns renewal scheduling and the client.

## RPCX

Only authentication v2 is accepted. `InitRpcxAuthV2` requires an explicit shared ReplayStore;
old initialization without one returns an error. Legacy MD5/nonce-less requests fail closed.
Signing occurs in a native SDK ClientBeforeEncode plugin AFTER argument serialization.
The signature covers version, domain, appId, service/method, timestamp, nonce and payload
SHA-256 using length-prefixed fields. Verification precedes atomic nonce admission.

The gokit adapter signs `appName/env/servicePath`; cross-project callers must explicitly
configure the target domain. Configure client plugins before first use, and never mutate
credential/plugin state while a client is active. Native retries can safely get fresh nonces;
this does not make a non-idempotent RPC safe to retry after an ambiguous response.
Shared replay storage unavailable => reject, no instance-local fallback. Default skew2m,
maximum10m. Credentials and replay Redis clients belong to the application lifecycle.

Native HTTP/JSON-RPC bridges are disabled. A mux read budget bounds handshake sniffing.
The SDK's process-wide frame limit is installed once at8MiB, with bounded gzip inflation
and owned compression buffers. Do not mutate protocol limits/compressors during operation.
TLS/mTLS, network policy and business authorization remain deployment/application concerns.

## Logging and HTTP

HTTP logging records route patterns/status/duration/length, not raw URL/query/header/body.
Recovery records panic type/stack, not panic values or credentials. A provided CheckError
still classifies a bounded response snapshot without writing it to logs. RPC wrappers no
longer dump args/reply. Applications must not deliberately pass secrets to general logger APIs.
HTTP servers distrust forwarding proxies by default; explicitly configure trusted proxies
through the Gin engine when a deployment needs them.
Logger reconfiguration/write/close are synchronized; replaced file sinks are closed. The
application closes the logger last; late logs after Close are no-ops.

`uHTTP.NewClient` owns a reusable bounded Transport. Default helpers share a default client;
custom per-call Config creates an owned short-lived client and closes it. Defaults: response
8MiB, upload64MiB, idle timeout90s, idle connections100, per-host idle10/connections100.
Responses reject oversize bodies; regular-file uploads stream with a byte budget. Do/Upload
honor context; CloseContext cancels active operations and waits for completion within budget.
Caller-provided blocking Reader implementations still must cooperate with cancellation.

## Reliable Redis/Delay

Default Worker deliveryMode is `reliable`; only explicit `legacy` uses destructive BLPOP /
ZREM plus process-local recovery. Do not run legacy consumers alongside reliable consumers
on the same source/failure pipeline during migration.

`taskqueue.Queue` accepts existing raw List or delayed ZSet producers. Claim moves payload
into persistent data/owner/visibility state atomically before returning it. Expired tasks are
recovered on subsequent claims. Lease renewal runs while business work is in flight. ACK,
retry and failure transfer require the current unexpired token; stale owners cannot delete
a successor's task. `taskqueue.TaskID(ctx)` exposes the same ID after recovery.

For newly produced reliable tasks, use Queue.Enqueue with a stable message ID. Zero due means
ready immediately; a future due schedules independently of payload contents. Duplicate pending
IDs are rejected, not overwritten. VisibilityTimeout defaults30s and is at least3*operationTimeout.
Business callbacks get cooperative handler deadlines; default delivery is at-least-once.
SIGKILL between side effect and ACK can repeat the side effect: handlers must be idempotent.
FailCount bounds local retries per delivery, not exactly the number of business executions
across every crash. Attempt records delivery count. A restarted consumer resumes stored work.

Failure transfer atomically retains ID/payload in the destination reliable queue. Two equal
payloads do not collapse as ZSet members. No failure destination => internal persistent dead
letter, never implicit discard. DeadLetters/RequeueDead/DeleteDead and Stats support operations.
The raw source key alone no longer represents total backlog; use Stats (including LegacySource).
Do not manually edit private `gokitbox:tasks:v1:*` keys. Pending/dead records have no expiry;
monitor backlog and apply explicit retention/dead-letter operations.

This module supports standalone Redis through *redis.Client, not Redis Cluster. Atomic scripts
use multiple keys. Durability requires appropriate Redis persistence/replication and no eviction
of task state; acknowledged storage loss cannot be repaired by the client library. The original
producer.Delay uses payload-as-member semantics; use Queue.Enqueue for distinct, stable jobs.
Kafka producer nil remains LOCAL acceptance, not broker confirmation, business completion or
transactional outbox. Application DB/MQ atomicity requires an application-specific outbox design.

## Other contracts and verification

SignByAscii now uses canonical v2 percent encoding and a protocol prefix; old integrations
must explicitly name SignLegacyByAscii. Event dispatch snapshots handlers under a lock and
calls them unlocked; listener changes affect the next dispatch. Callback panic propagates
explicitly. Event.Clone is a shallow clone including Object.

DB/Redis initialization and shutdown are serialized and terminal close prevents resurrection.
Preferred lookups use error-returning APIs; legacy nil/panic bridges remain documented.

Run `python3 scripts/verify-foundation.py` for a disposable clean-source build/vet/race run.
See verification/README.md for the versioned artifact and source binding. Unit/integration
checks do not substitute for real topology, failover durability or long-running load tests.
