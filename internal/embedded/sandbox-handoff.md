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

## See questions addressed to you (still open)

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

## Poll for incoming questions with /loop

Watch for questions addressed to you and answer them:

```
/loop 30s curl -sS "$BLVCKHOLE_HANDOFF_URL/handoff/threads?to=$BLVCKHOLE_SANDBOX&status=open"
```

When a thread appears, read it, do the work, and POST your answer to its
`/messages` endpoint. Stop the loop when the user no longer needs it.
