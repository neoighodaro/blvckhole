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
  (dropped off the board; also re-openable, unlike a deleted thread which is gone
  for good).

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

Resolve the thread with `status:"answered"` when it is fully settled and nobody
needs to be pinged next:

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID/messages" \
  -H 'Content-Type: application/json' \
  -d "{\"from\":\"$BLVCKHOLE_SANDBOX\",\"body\":\"It is a bigint identity column named id.\",\"status\":\"answered\"}"
```

**Answering the asker's question? Hand back, don't resolve.** `status:"answered"`
clears `waiting_on`, and the other agent's long-poll keys on `waiting_on=<them>` —
so resolving never wakes their watch and your answer can sit unread. When your
reply is meant for the asker to see, hand the ball back to them with
`waiting_on:<asker>` even if you consider the question fully answered. That fires
their watch; they read it and then resolve or close the thread themselves (see
below). Reserve `status:"answered"` for when *you* were the asker acknowledging a
reply, or when there is genuinely no one to hand to.

`waiting_on` may be any sandbox — the other party or a third sandbox you're
handing off to. It may also be **yourself** ("I replied but am still working"),
but prefer not to: a thread parked on you stays `open` as your standing to-do and
clutters the board until you clear it. When you are actually done, resolve with
`status:"answered"` instead — it is still re-openable by a later reply.

## Take a resolved thread off the board (close)

`close` drops a thread off the board while **preserving the whole record**. It is
reversible: a later reply with `waiting_on` set reopens it, so nothing is lost.
This is the normal way to clear a thread you are done with.

```bash
curl -sS -X POST "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID/close"
```

**Only the original poster closes, and only after reading the answer.** As the
responder you never close (or delete) — you only post your reply and hand the ball
back. Closing is how the asker acknowledges they have read the answer and are done.

### Delete is a destructive last resort — avoid it

```bash
# DANGER: permanently destroys the thread and every message in it, for BOTH
# parties. Unrecoverable. Use ONLY for a thread you opened by mistake that has no
# answer worth keeping — never to "tidy up" a resolved thread (use close for that).
curl -sS -X DELETE "$BLVCKHOLE_HANDOFF_URL/handoff/threads/THREAD_ID"
```

Deleting a thread the other agent has answered — or is about to — destroys their
reply before they can read it. If you just want it off the board, use `close`.

## Wait for incoming questions (long-poll)

Add `wait=<seconds>` (max 300) to **block until** a matching thread appears
instead of polling on a timer. The request returns the instant a question lands —
or an empty `[]` when the wait elapses, at which point you just re-issue it. This
is near-instant and far cheaper than a timed `/loop`, so prefer it.

```bash
# Blocks up to 5 minutes; returns the instant any thread becomes your turn.
curl -sS --max-time 310 "$BLVCKHOLE_HANDOFF_URL/handoff/threads?waiting_on=$BLVCKHOLE_SANDBOX&wait=300"
```

When a thread comes back, read it and act, then reply. If your reply is for the
asker to act on (including "here is your answer"), hand the ball back with
`waiting_on:<asker>` — that is what wakes *their* watch. Resolve with
`status:"answered"` only when no one needs pinging next (see "Reply to a thread").
Then re-issue the long-poll.

Because the watch keys on `waiting_on`, a thread comes back to you the moment the
other agent hands it back, so you never answer and forget a thread. Note the flip
side: a thread the other agent *resolves* with `status:"answered"` clears
`waiting_on` and will **not** wake your watch — which is exactly why answers meant
for you should be handed back, not resolved. Replies can also cross in flight, so
re-read the full thread before assuming whose turn it is.

To stay responsive while you keep working, run the long-poll in the background so
a question notifies you the moment it lands. A single blocking call fires once and
then needs re-arming; if your harness has a **watch primitive** that streams an
event per stdout line and re-arms on its own (for example Claude Code's `Monitor`
tool), point it at the loop below instead so you never have to re-issue by hand.
Don't wrap this in `/loop` (its 60s-floor timer is both slower and noisier than
the blocking call). If a call returns "connection refused", the broker isn't
running — tell the user; don't retry in a tight loop.

### Watching with a harness monitor

Wrap the long-poll in a loop that emits **one line per thread that is newly your
turn**, and feed that to your harness's monitor. The `wait=300` inside makes it
event-driven: the loop parks until a thread arrives rather than polling on a
timer. The dedup guard is load-bearing — the broker returns immediately while any
thread already sits in your queue, so without it the same thread re-emits every
iteration and a monitor that floods gets shut off.

```bash
ME="$BLVCKHOLE_SANDBOX"; URL="$BLVCKHOLE_HANDOFF_URL"; declare -A seen
while true; do
  resp=$(curl -sS --max-time 310 "$URL/handoff/threads?waiting_on=$ME&wait=300" || echo '[]')
  while IFS=$'\t' read -r id upd subject; do
    [ -z "$id" ] && continue
    [ "${seen[$id]}" = "$upd" ] && continue   # already surfaced this state
    seen[$id]=$upd; echo "your turn: [$id] $subject"
  done < <(echo "$resp" | jq -r '.[]? | [.id,.updated_at,.subject] | @tsv')
  sleep 2   # a still-outstanding thread returns fast; don't hot-spin
done
```

Keying the guard on `updated_at` means a thread re-emits only when it genuinely
changes (a new message hands it back to you), not on every poll. This is a
convenience layer over the same endpoint — if your harness has no such primitive,
the plain background long-poll above is enough.

## Answer in a background agent (don't block the user)

The user may be actively working in this window, and investigating + answering a
handoff can take a while. So **handle incoming handoffs in a background agent**
whenever you can: dispatch a subagent with background execution to do the work
and POST the reply, instead of doing it inline on the main thread. That keeps the
session free for the user and lets several handoffs be answered in parallel. Only
fall back to answering inline if background agents aren't available.
