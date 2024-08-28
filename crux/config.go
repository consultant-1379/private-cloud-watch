package version

import "strings"

// DO NOT EDIT THIS SECTION!
// These build configuration strings that are used
// to determine the provenance of the particular build of this binary.
// These configuration strings are set using the LDFlags settings.
// go build -X main.myriadCommitID=foo
// LDFlags are defined using the ./tools/gen-ldflags.go script.
var (
	cruxBuildDatetime  = "UNKNOWN"
	cruxCommitDatetime = "$Format:%cI$"
	cruxCommitID       = "$Format:%H$"
	cruxShortCommitID  = "$Format:%h$"
	cruxGitTreeIsClean = "UNKNOWN"
	cruxBranchName     = "UNKNOWN"
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

	// If there is no previously tagged commit then the
	// gitTag will be empty and this is not an official release.
	if len(gitTag()) == 0 {
		return false
	}

	// gitTag will contain a '-d+' indicating the number of
	// commits since the previous tag. idx == -1 if there are
	// no dashes, and thus this is a tagged commit.
	if idx := strings.IndexByte(gitTag(), '-'); idx == -1 {
		return true
	}
	return false
}
