package confine

import (
	"path"
	"strings"
)

// floorPatterns is the non-configurable sensitive-file floor
// (specs 10 — Repository confinement, specs 14 — Security and data exposure).
// It is a floor *inside* allowed subtrees, not the boundary — the allow-list
// is the boundary — and it exists for the common case of `--allow .`.
//
// Patterns are matched with path.Match against every segment of a wire path,
// not only its last: a directory called `credentials` hides credentials just
// as surely as a file called `credentials.json`, and `.git` is on the list as
// a segment precisely so that everything beneath it is refused. Repository
// history has a proper tool (git_read), and `.git/config` carries credential
// helper configuration and remote URLs that can contain tokens.
//
// Nothing here is configurable. A caller who could widen the floor could
// widen it to nothing, and the point of a floor is that `--allow .` still
// cannot read a credential.
var floorPatterns = []string{
	".env*",
	"*.pem",
	"*.key",
	"id_rsa*",
	"*.p12",
	"*.pfx",
	".npmrc",
	".netrc",
	"credentials*",
	".git",
}

// deniedByFloor reports whether any segment of a cleaned wire path matches the
// floor.
//
// Matching folds case. Windows folds it in the filesystem whatever this code
// does, so a case-sensitive floor would refuse `.env` on Linux and `.ENV`
// nowhere — the same repository readable on one matrix OS and not the other.
// Folding here costs an over-refusal of a Linux file spelled `NOTES.KEY`,
// which is the safe direction of that trade.
func deniedByFloor(wirePath string) bool {
	for _, segment := range strings.Split(wirePath, "/") {
		folded := strings.ToLower(segment)
		for _, pattern := range floorPatterns {
			if ok, err := path.Match(pattern, folded); err == nil && ok {
				return true
			}
		}
	}
	return false
}
