# Issue 09 — hand-run SIGINT verification

Acceptance criterion: `external-reviewer review <repo>` with stdin on a terminal
terminates on Ctrl-C with a `done` line and exit `2`. Delivering a real `SIGINT` is not
portable to the Windows CI runner or reliably scriptable on Windows at all, so this was
verified against a real Linux kernel: `wsl -d docker-desktop`, the same environment
issue 10 of this epic used for its SIGPIPE criterion.

## Setup

The Windows host has no native way to deliver a real `SIGINT`; `wsl -d docker-desktop`
exposes a genuine Linux kernel where `kill -INT` behaves exactly as a shell's Ctrl-C
does. The `docker-desktop` distro's root filesystem is small (~136 MB, ~54 MB free at
the time of this run), so only the one binary needed was copied in.

```
GOOS=linux GOARCH=amd64 go build -o external-reviewer-linux .   # on the Windows host
wsl -d docker-desktop -- cp /mnt/host/c/.../external-reviewer-linux /tmp/external-reviewer
wsl -d docker-desktop -- chmod +x /tmp/external-reviewer
```

Stdin was attached to a named pipe (fifo) opened for both read and write (`exec
3<>/tmp/stdin_fifo`), rather than a real pty: this leaves stdin open, attached and
producing no bytes — the same state a terminal is in before a human has typed
anything — without requiring an interactive pty allocation through the wsl.exe
non-interactive invocation path. The process was then interrupted with a real `SIGINT`
via `kill -INT`, not a context cancellation constructed in test code.

## Script

```sh
mkdir -p /tmp/repo
rm -f /tmp/stdin_fifo
mkfifo /tmp/stdin_fifo
exec 3<>/tmp/stdin_fifo
/tmp/external-reviewer review /tmp/repo <&3 >/tmp/stdout.log 2>/tmp/stderr.log &
PID=$!
echo "started pid=$PID"
sleep 1
ls /proc/$PID 2>&1 | head -1     # confirm the process is alive and blocked
kill -INT $PID
echo "kill rc=$?"
sleep 1
ls /proc/$PID 2>&1 | head -1     # confirm the process is gone
wait $PID
echo "exit_code=$?"
echo "--- stdout ---"
cat /tmp/stdout.log
echo "--- stderr ---"
cat /tmp/stderr.log
```

## Transcript (actual output)

```
started pid=58
arch_status
kill rc=0
ls: /proc/58: No such file or directory
exit_code=2
--- stdout ---
--- stderr ---
error:  run interrupted
done    turns=0  in=0 out=0  $0.0000  1.0s  stop=interrupted
```

## Reading the transcript

- `arch_status` (the first line inside `/proc/58/`, listed via `head -1`) confirms the
  process was alive and blocked in the stdin read one second after starting — it had
  not exited on its own.
- `kill -INT $PID` returned `0`: the signal was delivered.
- The second `/proc/$PID` check, one second later, shows the process is gone —
  `kill -INT` alone does not remove a process; `run`'s own return path does, through the
  deferred `diag.WriteDone` in `internal/cli/run.go`.
- `wait $PID` reports `exit_code=2` — SPECS' exit-2 classification, not termination by
  signal (which `wait` would report as a value ≥ 128, e.g. 130 for a killed process
  never reaching `os.Exit` at all — the exact failure mode this issue exists to close).
- `stdout` is empty.
- `stderr` carries exactly one `error:` line (`run interrupted`) and one `done` line
  with `stop=interrupted`, `turns=0`, `in=0 out=0` — the run never reached a model,
  because it was still blocked reading the prompt from stdin.

This is the same outcome
`TestRun_Review_StdinCancelledMidRead_ExitsInterrupted` (internal/cli/stdin_test.go)
asserts through a constructed, pre-cancelled context rather than a real signal — this
file is the by-hand confirmation that `run.go`'s real `signal.NotifyContext` wiring
carries a real `SIGINT` into the same path.
