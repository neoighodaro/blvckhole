---
name: sandbox-handoff
description: Use to ask another blvckhole sandbox a question and get an answer, or to answer questions other sandboxes addressed to you. Cross-sandbox threaded question/answer handoff over the shared blvckhole handoff broker. Triggers: handoff, ask another sandbox, cross-sandbox question, answer another agent.
---

# Sandbox Handoff

Exchange threaded questions and answers with other blvckhole-managed sandboxes
through a shared broker running on the host.

- Your identity (`from`) is **`$BLVCKHOLE_SANDBOX`**.
- The broker base URL is **`$BLVCKHOLE_HANDOFF_URL`** (e.g. `http://host.docker.internal:8787`).
- Address other sandboxes by their **sandbox name** (the `name` in their `blvckhole.yaml`).

> The broker only runs while the user has started it on the host with
> `blvckhole handoff`. If a call fails with "connection refused", the broker is
> not running — tell the user; do not retry in a tight loop.

## A thread

`{ id, from, to, subject, status, created_at, updated_at, messages[] }` where
`status` is `open` (awaiting an answer) or `answered`. Status flips
automatically: when the recipient replies it becomes `answered`; when the
original asker follows up it returns to `open`. You never set status yourself.

## Ask another sandbox a question (open a thread)

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads" \
  -H 'Content-Type: application/json' \
  -d "{\"from\":\"$BLVCKHOLE_SANDBOX\",\"to\":\"other-sandbox\",\"subject\":\"DB schema\",\"body\":\"What is the users table primary key?\"}"
```

## See questions addressed to you (immediate snapshot)

Returns right away — use it for a one-shot check (e.g. at the start of a session):

```bash
curl -sS "$BLVCKHOLE_HANDOFF_URL/handoff/threads?to=$BLVCKHOLE_SANDBOX&status=open"
```

## Read one thread

```bash
curl -sS "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID"
```

## Answer a thread (or follow up)

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID/messages" \
  -H 'Content-Type: application/json' \
  -d "{\"from\":\"$BLVCKHOLE_SANDBOX\",\"body\":\"It is a bigint identity column named id.\"}"
```

## Close a thread you opened

```bash
curl -sS -X DELETE "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID"
```

## Wait for incoming questions (long-poll)

Add `wait=<seconds>` (max 300) to **block until** a matching thread appears
instead of polling on a timer. The request returns the instant a question lands —
or an empty `[]` when the wait elapses, at which point you just re-issue it. This
is near-instant and far cheaper than a timed `/loop`, so prefer it.

```bash
# Blocks up to 5 minutes; returns immediately when a question for you arrives.
curl -sS --max-time 310 "$BLVCKHOLE_HANDOFF_URL/handoff/threads?to=$BLVCKHOLE_SANDBOX&status=open&wait=300"
```

When a thread comes back, read it, do the work, POST your answer to its
`/messages` endpoint, then re-issue the long-poll to keep listening.

To stay responsive while you keep working, run the long-poll as a **background
task** — you'll be notified the moment it returns with a question, and you can
re-arm it after answering. Don't wrap this in `/loop` (its 60s-floor timer is
both slower and noisier than the blocking call). If a call returns "connection
refused", the broker isn't running — tell the user; don't retry in a tight loop.
