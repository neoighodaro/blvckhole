---
name: sandbox-handoff
description: Use to ask another blvckhole sandbox a question and get an answer, or to answer questions other sandboxes addressed to you. Cross-sandbox threaded question/answer handoff over the shared blvckhole handoff broker. Triggers: handoff, ask another sandbox, cross-sandbox question, answer another agent.
---

# Sandbox Handoff

Exchange threaded questions and answers with other blvckhole-managed sandboxes
through a shared broker running on the host.

Reach for this proactively: when something you're working on depends on another
sandbox's domain — an interface or contract, a data shape, the meaning of a
field, expected behavior — and it's ambiguous or you'd otherwise guess, open a
thread and ask instead of assuming. A quick question beats a wrong assumption.

- Your identity (`from`) is **`$BLVCKHOLE_SANDBOX`**.
- The broker base URL is **`$BLVCKHOLE_HANDOFF_URL`** (e.g. `http://host.docker.internal:8787`).
- Address other sandboxes by their **sandbox name** (the `name` in their `blvckhole.yaml`).

> The broker only runs while the user has started it on the host with
> `blvckhole handoff`. If a call fails with "connection refused", the broker is
> not running — tell the user; do not retry in a tight loop.

## A thread

`{ id, from, to, subject, status, waiting_on, created_at, updated_at, messages[] }`.

- `waiting_on` names the sandbox **whose turn it is** — the one who should act
  next. Watch for threads where `waiting_on` is you (see below).
- `status` describes resolution: `open` (in flight, someone's turn), `answered`
  (resolved, nobody's turn — still re-openable by a later reply), or `closed`
  (dropped off the board).

Turn-taking is **explicit**: every reply must say what happens next. There is no
automatic flip — *you* decide whether you're handing the ball back or resolving
the thread. Opening a thread sets `waiting_on` to the recipient automatically.

## Ask another sandbox a question (open a thread)

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads" \
  -H 'Content-Type: application/json' \
  -d "{\"from\":\"$BLVCKHOLE_SANDBOX\",\"to\":\"other-sandbox\",\"subject\":\"DB schema\",\"body\":\"What is the users table primary key?\"}"
```

## See threads where it's your turn (immediate snapshot)

Returns right away — use it for a one-shot check (e.g. at the start of a session).
Filtering on `waiting_on` covers **both directions**: threads others addressed to
you *and* threads you opened that someone has handed back to you.

```bash
curl -sS "$BLVCKHOLE_HANDOFF_URL/handoff/threads?waiting_on=$BLVCKHOLE_SANDBOX"
```

## Read one thread

```bash
curl -sS "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID"
```

## Reply to a thread — always state intent

Every reply must include **either** `waiting_on` (hand the ball to whoever should
act next) **or** `status:"answered"` (resolve the thread). A reply with neither —
or both — is rejected. This is deliberate: it stops you from accidentally marking
a thread "answered" when you actually need the other agent to clarify.

Hand the ball back (e.g. you answered but need a clarification before you can
finish):

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID/messages" \
  -H 'Content-Type: application/json' \
  -d "{\"from\":\"$BLVCKHOLE_SANDBOX\",\"body\":\"Got it — but which environment?\",\"waiting_on\":\"other-sandbox\"}"
```

Resolve the thread when everything is settled (either party may do this):

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID/messages" \
  -H 'Content-Type: application/json' \
  -d "{\"from\":\"$BLVCKHOLE_SANDBOX\",\"body\":\"It is a bigint identity column named id.\",\"status\":\"answered\"}"
```

`waiting_on` may be any sandbox — the other party, **yourself** (you replied but
are still working and will follow up), or a third sandbox you're handing off to.

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
# Blocks up to 5 minutes; returns the instant any thread becomes your turn.
curl -sS --max-time 310 "$BLVCKHOLE_HANDOFF_URL/handoff/threads?waiting_on=$BLVCKHOLE_SANDBOX&wait=300"
```

When a thread comes back, read it and act. If you can resolve it, reply with
`status:"answered"`. If you need something first, reply with `waiting_on` set to
whoever should act next — then re-issue the long-poll. Because the watch keys on
`waiting_on`, a thread you opened comes back to you the moment the other agent
hands it back, so you never answer and forget a thread.

To stay responsive while you keep working, run the long-poll as a **background
task** — you'll be notified the moment it returns with a question, and you can
re-arm it after answering. Don't wrap this in `/loop` (its 60s-floor timer is
both slower and noisier than the blocking call). If a call returns "connection
refused", the broker isn't running — tell the user; don't retry in a tight loop.

## Answer in a background agent (don't block the user)

The user may be actively working in this window, and investigating + answering a
handoff can take a while. So **handle incoming handoffs in a background agent**
whenever you can: dispatch a subagent with background execution to do the work
and POST the reply, instead of doing it inline on the main thread. That keeps the
session free for the user and lets several handoffs be answered in parallel. Only
fall back to answering inline if background agents aren't available.
