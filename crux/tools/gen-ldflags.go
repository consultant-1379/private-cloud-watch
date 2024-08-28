// +build ignore

/*
gen-ldflags.go

Generate linker flags containing version and release metadata for
go binaries. Run this go source using `go run` from the root of
the repository to generate a file "ldflags" that contains the compiler
flags with release metadata. Then use the "ldflags" file as a make
target for build and install steps.  While the metadata includes a
"cruxReleaseDatetime", the ldflags file is not always updated. This
command will leave the existing ldflags file alone if the following
conditions are met:

 - git repository is clean.
 - Branch name is unchanged or CRUX_RELEASE is set.
 - CRUX_RELEASE env variable is unchanged (may be unset).
 - Commit SHA and short-SHA are unchanged.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

// genLDFlags generates LD flag command string for go compiler
func genLDFlags(datetime string) string {
	ldFlagStr := ""
	ldFlagStr = ldFlagStr + "-X version.cruxBuildTag=" + releaseTag() + " "
	ldFlagStr = ldFlagStr + "-X version.cruxBuildDatetime=" + datetime + " "

	if !missingGit {
		cdatetime := cmdExec("git", "log", "--format=%cI", "-n", "1", "HEAD")
		ldFlagStr = ldFlagStr + "-X version.cruxCommitDatetime=" + cdatetime + " "
		ldFlagStr = ldFlagStr + "-X version.cruxCommitID=" + commitID() + " "
		ldFlagStr = ldFlagStr + "-X version.cruxShortCommitID=" + shortCommitID() + " "
		ldFlagStr = ldFlagStr + "-X version.cruxBranchName=" + branchName() + " "
		ldFlagStr = ldFlagStr + "-X version.cruxGitTreeIsClean=" + fmt.Sprintf("%t", isClean()) + ""
	}
	return ldFlagStr
}

// getState returns a string containing all the things in genLDFlags other
// than the date. The state string is used to determine whether the ldflags
// file needs to be updated--and thus cause Make to recompile and reinstall
// all crux binaries.
func getState() (state string) {
	state += commitID()
	state += shortCommitID()
	if isRelease() {
		state += releaseTag()
	} else {
		state += "branch-"
		state += branchName()
	}
	return
}

// genReleaseTag prints release tag to the console for easy git tagging.
func releaseTag() string {
	if tag := os.Getenv("CRUX_RELEASE"); tag != "" {
		return tag
	}
	return "UNOFFICIAL"
}

// isRelease returns true if CRUX_RELASE is set.
func isRelease() bool {
	if tag := os.Getenv("CRUX_RELEASE"); tag != "" {
		return true
	}
	return false
}

// commitID returns the SHA1 of HEAD.
func commitID() string {
	// git rev-parse HEAD
	return cmdExec("git", "rev-parse", "HEAD")
}

// shortCommitID returns the shortened SHA1 of HEAD
func shortCommitID() string {
	// git rev-parse --short HEAD
	return cmdExec("git", "rev-parse", "--short", "HEAD")
}

// branchName returns the current branch name
func branchName() string {
	// git rev-parse --abbrev-ref HEAD
	return cmdExec("git", "rev-parse", "--abbrev-ref", "HEAD")
}

// isClean returns true if working tree is clean
// which means no staged or unstaged changes.
func isClean() bool {
	if missingGit {
		return false
	}

	// git diff --quiet && git diff --cached --quiet
	err := exec.Command("git", "diff", "--quiet").Run()
	if err != nil {
		return false
	}
	err = exec.Command("git", "diff", "--cached", "--quiet").Run()
	if err != nil {
		return false
	}
	return true
}

// cmdExec helper to execute a command and return the output.
func cmdExec(cmd ...string) string {
	args := []string{}
	if len(cmd) > 1 {
		args = cmd[1:]
	}
	out, err := exec.Command(cmd[0], args...).Output()
	if err != nil {
		msg := "Error executing command '%s'\nError: %s"
		exitErr(1, msg, strings.Join(cmd, " "), err)
	}
	return strings.TrimSpace(string(out))
}

// exitErr helper function to print error to stderr and exit with return code.
func exitErr(code int, fmtstr string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmtstr, args...)
	os.Exit(code)
}

// The .ldflags file is where we retain the "state" data for the current
// ldflags file. This old state is compared to the current getState() to
// determine if we should update the ldflags file or not.
const ldFlagStateFile = ".ldflags"

var missingGit bool

func main() {
	if len(os.Args) < 2 {
		exitErr(1, "Usage: go run ./tools/gen-ldflags.go <outfile>")
	}

	// Just allow make to continue if this information is not available.
	_, err := exec.Command("git", "-C", ".", "rev-parse").Output()
	if err != nil {
		msg := "gen-ldflags.go: Not a git repository. Omitting some --version information.\n"
		fmt.Fprint(os.Stderr, msg)
		missingGit = true
	}

	ldFlagsExists := false
	if _, err := os.Stat(os.Args[1]); err == nil {
		ldFlagsExists = true
	}

	// don't bother comparing old to current state if working tree is
	// unclean or we are missing the final 'ldflags' file.
	if ldFlagsExists && isClean() {
		state := getState()
		stateFile, err := os.Open(ldFlagStateFile)
		if err == nil {
			oldState, err := ioutil.ReadAll(stateFile)
			if err != nil {
				exitErr(1, "%v", err)
			}

			stateFile.Close()

			// Working tree is clean and state is unchanged. Don't
			// update anything.
			if string(oldState) == state {
				return
			}

			// If we got here state is different, so remove the state file.
			err = os.Remove(ldFlagStateFile)
			if err != nil {
				exitErr(1, "%v", err)
			}
		}

		// Now write state file, either it didn't exist or we just deleted
		// it because the state changed.
		ioutil.WriteFile(ldFlagStateFile, []byte(state), 0666)
	}

	// Open and write the flags to the 'ldflags' file now.
	file, err := os.OpenFile(os.Args[1], os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		exitErr(1, "Error opening file for output: %v", err)
	}
	defer file.Close()
	// date is in RFC3339 TimeFormat.
	datetime := time.Now().UTC().Format(time.RFC3339)
	fmt.Fprintln(file, genLDFlags(datetime))
}
