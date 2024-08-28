package devtools

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// The Mark*() functions are intended as an improvement over the
// printf's often used for temporary debugging in code under
// development.  Marks' output the line number and function name of
// the calling function.  Also, you can group marks into "trails" for
// selective output.

// "Trails" are intended to separate specific aspects or paths through
// your code, and allow easy selective output for those areas.

// To disable a trail for a single package, use an inline TrailOff(),
// or for all packages, create a Env var named <name>-TRAIL. The
// var's value is not used.

// Ex: os.Setenv("DATA-TRAIL","foo")

// FirstMark - Set to false if you don't want a separator line between
// runs.  FIXME: Not global state, so every package will output a
// separator line.  Okay for simple package level testing...
var FirstMark = true

// TrailsOff - quick way to disable all output.
const TrailsOff = false //TODO: allow setting via an env var.

var hiddenTrails = make(map[string]bool)

// DefaultTrail - name to use when user doesn't specify their own "trail".
const DefaultTrail string = "DFLT"

// Some generic trail names likely to be applicable in many programs.
//  DATA, LIFECYC, IO, MEM, TRANSFORM, LOOKUP, DBASE,

// TrailOn - Output for marks on this trail
func TrailOn(trail string) {
	hiddenTrails[trail] = false
}

// TrailOffEnv - Turn off a trail for all packages.  Must be called
// before the the first execution of a mark for this trail occurs.
func TrailOffEnv(trail string) {
	os.Setenv(trail+"-TRAIL", "foo")
}

// TrailOff - No output for marks on this trail
func TrailOff(trail string) {
	hiddenTrails[trail] = true
}

// Mark - Mark using the default trail. Like Printf with the calling
// function name and line number.
// FIXME: too easy to use Mark() with a traillname.  names not typed so no error.
// FIXME: Add \n , create Markf()
func Mark(format string, args ...interface{}) {
	trail := DefaultTrail // TODO: make function name the default trail.
	mark(trail, format, args...)
}

// MarkT - Mark using a specific trail.
func MarkT(trail string, format string, args ...interface{}) {
	// Straight pass-through, to get stack depth right for Caller()
	mark(trail, format, args...)
}

func mark(trail string, format string, args ...interface{}) {
	trailOn := true
	var foundTrail bool
	// First mark*() for a trail lazily sets on/off if not already
	// set by TrailOn/TrailOff. Gets set to ON unless there's and
	// an environment variable with the correct name.
	//
	// When disabling via env vars, be sure to create the var
	// before the first trail mark is encountered, or it will have
	// no effect.
	trailOn, foundTrail = hiddenTrails[trail]

	// Visually separates runs' output.
	// However, the program itself could output before the 1st mark.
	if FirstMark {
		fmt.Printf("\n\n============= FIRST MARK ========================\n")
		FirstMark = false
	}

	if trailOn, foundTrail = hiddenTrails[trail]; !foundTrail {
		_, foundEnv := os.LookupEnv(trail + "-TRAIL")
		hiddenTrails[trail] = !foundEnv
		trailOn = !foundEnv

	}

	if !trailOn || TrailsOff {
		return
	}

	// Trail is on, go ahead and output

	pc, _ /*file*/, line, _ := runtime.Caller(2)

	//  Turn this: "github.com/erixzone/crux/pkg/walrus.(*Entry).Log"
	//  Into this: "Log"
	funcName := runtime.FuncForPC(pc).Name()
	sl := strings.Split(funcName, ".")
	funcName = sl[len(sl)-1]

	info := fmt.Sprintf(format, args...)
	trailPrefix := "Trail"
	fmt.Printf("<%s %s>: %s() %d>  %s\n", trailPrefix, trail, funcName, line, info)
}
