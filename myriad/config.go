package main

import "strings"

// DO NOT EDIT THIS SECTION!
// These build configuration strings that are used
// to determine the provenance of the particular build of this binary.
// These configuration strings are set using the LDFlags settings.
// go build -X main.myriadCommitID=foo
// LDFlags are defined using the ./tools/gen-ldflags.go script.
var (
	myriadBuildDatetime  = "UNKNOWN"
	myriadCommitDatetime = "2019-07-31T14:22:16-04:00"
	myriadCommitID       = "97fb5aab142f30ddf19ce49be17866685ab6e4b4"
	myriadShortCommitID  = "97fb5aa"
	myriadBranchName     = "UNKNOWN"
	myriadGitTreeIsClean = "UNKNOWN"
)

// DO NOT EDIT: this is set by filter scripts in the local git
// repository. Run 'make gitconfig' to register these filter scripts.
// See ./tools/git-filters/tagger/README.md for documentation.
var gitTagVal = "=GIT_TAG:="

// gitTag removes the cruft around the tag, reducing it to
// the output of running `git describe --tag HEAD` on the repo
// at checkout of the current HEAD.
func gitTag() string {
	t := strings.TrimSuffix(gitTagVal, "=")
	return strings.TrimPrefix(t, "=GIT_TAG:")
}

func isOfficial() bool {
	// gitTag will contain a '-d+' indicating the number of
	// commits since the previous tag. idx == -1 if there are
	// no dashes, and thus this is a tagged commit.
	if idx := strings.IndexByte(gitTag(), '-'); idx == -1 {
		return true
	}
	return false
}
