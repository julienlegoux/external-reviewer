#!/usr/bin/env bash
#
# Run the decided test command -- `go test ./... -race` -- on a remote Linux host.
#
# The development machine is Windows and runs `go test ./...` natively -- but not
# with `-race`, which needs cgo, and the machine's Application Control policy refuses
# the unsigned DLLs a Windows C toolchain loads at startup. So the plain suite stays
# local and fast, and `-race` comes here. See docs/planning/DRIFT.md and
# docs/planning/CONVENTIONS.md section Testing.
#
# The remote needs three things: a Go toolchain, a C compiler, and an ssh entry.
# Nothing is installed by this script and nothing is left running -- it copies the
# working tree, runs the command, and prints what the command printed.
#
# Usage, from anywhere inside the repository:
#
#   scripts/test-remote.sh                                  # the whole suite
#   scripts/test-remote.sh ./internal/cli                   # one package
#   scripts/test-remote.sh ./internal/cli -run TestFoo -v   # one test, verbose
#
# `-race` is appended by the script and is not optional -- running without it is
# what this script exists to stop being the only choice.
#
# Environment:
#   EXTERNAL_REVIEWER_TEST_REMOTE   ssh destination      (default: vps)
#   EXTERNAL_REVIEWER_TEST_DIR      remote scratch path  (default: ci/external-reviewer)

set -euo pipefail

remote="${EXTERNAL_REVIEWER_TEST_REMOTE:-vps}"
remote_dir="${EXTERNAL_REVIEWER_TEST_DIR:-ci/external-reviewer}"

cd "$(git rev-parse --show-toplevel)"

if [ "$#" -eq 0 ]; then
	set -- ./...
fi

# The remote login shell is not interactive, so a Go installed under /usr/local/go
# is not on PATH. Appending rather than prepending leaves a remote that manages its
# own toolchain in charge of which one wins.
remote_path='export PATH="$PATH:/usr/local/go/bin"'

quote() { printf '%q' "$1"; }

q_dir=$(quote "$remote_dir")

q_args=""
for arg in "$@" -race; do
	q_args="$q_args $(quote "$arg")"
done

# Preflight, so a missing toolchain is one clear line rather than a compiler error
# three minutes into a sync.
if ! ssh "$remote" "$remote_path; command -v go >/dev/null" >/dev/null 2>&1; then
	echo "test-remote: no Go toolchain on '$remote' (looked on PATH and in /usr/local/go/bin)" >&2
	echo "test-remote: install one there, or point EXTERNAL_REVIEWER_TEST_REMOTE at a host that has one" >&2
	exit 1
fi
if ! ssh "$remote" 'command -v cc >/dev/null || command -v gcc >/dev/null' >/dev/null 2>&1; then
	echo "test-remote: no C compiler on '$remote'; -race needs cgo" >&2
	echo "test-remote: apt-get install build-essential (or the host's equivalent)" >&2
	exit 1
fi

# The working tree as git sees it: tracked files plus untracked ones that are not
# ignored, which is what makes a red-green cycle on an uncommitted test possible.
# Files git still lists but that are gone from disk are dropped, so a deletion in
# progress does not abort the sync.
git ls-files --cached --others --exclude-standard -z |
	while IFS= read -r -d '' file; do
		if [ -e "$file" ]; then
			printf '%s\0' "$file"
		fi
	done |
	tar -czf - --null -T - |
	ssh "$remote" "rm -rf -- $q_dir && mkdir -p -- $q_dir && tar -xzf - -C $q_dir"

# The module cache and build cache live in the remote home, outside the directory
# wiped above, so only the first run pays for downloads and a cold build.
exec ssh "$remote" "$remote_path; cd $q_dir && go test$q_args"
