// (c) Ericsson Inc. 2019 All Rights Reserved
// Contributors:
//      Christopher W. V. Hogue

package auto

import (
	"math"
	"math/rand"
	"time"
)

// CONFIDENCE = The provided timeout in code arises from
// tested sets of conditions: Use ConfUNKNOWN or ConfLOW
// if you are likely to run this code and this timeout
// value on other types of machines and systems.
const (
	ConfPROD    = 6 // timeout known from lots of use in production
	ConfSTABLE  = 4 // timeout known from some statistics of running on test hardware
	ConfKNOWN   = 3 // timeout known on my development system
	ConfUNKNOWN = 2 // timeout is an educated guess
	ConfLOW     = 1 // timeout is just a guess
)

// TRANSIT = What this function's data and code traverses
// Use the appropriate constant to describe the tomaton task implementation:
const (
	TransitRAM  = 1 // Data and/or code working out of ram
	TransitSSD  = 2 // Data and/or code hitting ssd storage
	TransitRPC  = 3 // Data and/or code traversing a newtwork stack in one physical computer
	TransitLAN  = 4 // Data and/or code traversing the local network
	TransitDISK = 5 // Data and/or code hitting spinning hard disk
	TransitWAN  = 6 // Data and/or code executed somewhere out on the Internet
)

// TLDR; Quick Guide To Skeptical Timeouts:
// Kind of like a hockey referee who calls penalties according to
// strict rules in the regular season, but ignores a lot of calls
// in the playoffs to avoid stoppages in the game.
//
// -> A Hard timeout (original tomaton behavior):
// Conf = 0
// Transit = 0
// To = 500 * time.Millisecond
//
// -> Never timeout:
// Conf = 0
// Transit = 0
// To = 0 * time.Second
//
// -> A quick gRPC call across your own network, try:
// Conf = ConfUNKNOWN
// Transit = TransitLAN
// To = 500 * time.Millisecond
//
// -> A disk write (spinning rust), try:
// Conf = ConfUNKNOWN
// Transit = TransitDISK
// To = 1 * time.Second
//
// -> An SSD write, try:
// Conf = ConfUNKNOWN
// Transit = TransitSSD
// To = 500 * time.Millisecond
//

// Tomaton - Skeptical Timeouts
//
// Tomaton workers evaluates timeouts for tasks.
// The first version used strict, hard timeouts, which could not be ignored.
// As tomaton workflow gets ported to compute systems of varying speed, or workflow
// tasks take paths of uncertainty, hard coded timeouts become the major cause
// of workflow breakage.
//
// When there is no data to estimate timeouts, or the data arises from a
// particular developer's bespoke system, no matter how confident they are in
// the provided timeout, it can be too short on another system.
// It seems better to treat most timeouts with some
// degree of skepticism, especially when that timeout value is
// given the power to halt and fail workflow.
// Any unportable or overzealous value
// is going to behave as a bug, rather than a feature.
//
// Here, the evaluation of a timeout is treated with a hieuristic
// "skeptic" function, which returns true or false. The hieuristic is
// based on values provided by the progammer about 1) the code and
// data path of the workflow task with the "TRANIST_" constants.
// and 2) their confidence in the timeout as provided with the
// "Conf" connstants. See the bottom of this file for how the
// math works.
//
// These values should be easy to provide at develoment time
// and the timeout and information can be refined as more
// information becomes known about expected
// task timing behavior.
//
// A provided timeout is modified when Transit and Conf values
// are specified, with a hieuristic to estimate the "hardness"
// of the timeout.
//
// When Transit and/or Conf are left as 0, the timeout is considered
// a hard as-is timeout.
//
// Finally a timeout of 0 - now ignores the timeout completely, the
// timeout function is disabled and always returns false.

const zero int64 = 0
const one float64 = 1

var rnd *rand.Rand

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// SkepticalTimeout - takes the duration of timeout and elapsed time being
// considered, as well as the Conf and Transit values, and returns
// a boolean that calls whether the conditions are a really a timeout.
func SkepticalTimeout(to, elapsed time.Duration, conf, transit int) bool {
	toi := int64(to)
	eli := int64(elapsed)
	ot := eli - toi // overtime duration
	if ot <= zero || toi <= zero {
		return false // not a timeout, or no timeout value set
	}
	if conf == 0 || transit == 0 {
		return true // trigger hard timeout as ot is > 0
	}
	// Evaluate skeptic funciton p value
	tof := float64(to)
	cmf := float64(conf - transit)
	p := math.Log10((tof + float64(ot)*math.Exp2(cmf)) / tof)
	// fmt.Printf("p=%f\n", p)
	if p >= one {
		return true // definitely trigger timeout
	}
	// When p less than 1 , roll dice
	r := rnd.Float64() // random number from 0-1
	// fmt.Printf("r=%f\n", r)
	if r+p < one {
		return false // p-based timeout ignore
	}
	return true // p-based timeout trigger
}

//
// Understanding the timeout equation components:
// This is what I call the "skeptic" funciton, it is a hieuristic
// that behaves like a probability (although it may exceed 1).
//
//	p := Log10((to + ot * 2**(conf-transit)) / to)
//
// p is computed from the Log10 ratio of the elapsed time to the timeout
// where elapsed = (to + overtime).
// Also - we don't compute p when elapsed < timeout.
//
// So consider p computed with this simpler form:
//      p = Log10((to + ot)/to)
//
// p samples for a 1s timeout:
// 1/1       = 0
// 1.00001/1 = 0.0000043
// 1.001/1   = 0.00043
// 1.1/1     = 0.041
// 2/1       = 0.301
// 10/1      = 1
//
//
// So p is modified by the provided conf and transit values as follows:
// Conf and Transit const values are provided as base 2 exponents.
// These are used to scale the value of ot up or down
// via the factor 2**(conf-transit)
//
// Samples of (conf-transit) and 2**(conf-transit)
// conf - transit                 2**(conf-transit)
// 6 - 1  =  5 (prod, ram)        2**(5)  = 32       best case
// 5 - 2  =  3 (stable, ssd)      2**(3)  = 8
// 3 - 3  =  0 (known, rpc)       2**(0)  = 1
// 1 - 4  = -3 (low, lan)         2**(-3) = 0.125
// 1 - 6  = -5 (low, wan)         2**(-5) = 0.03125  worst case
//
// Thus 2**(conf-transit) provides a range of scaling of
// ot from 0.03125*ot  to 32*ot
//
// How much time may elapse before this behaves like a hard timeout?
// - Solve equation for 1s, where p = 1,
//   with upper and lower values of (conf-transit)
//   1s timeout:
//   log( (1 + 0.28125 * 32) / 1)   = 1   (elapsed time is 1.28125s)
//   log( (1 + 288 * 0.03125) / 1)  = 1   (elapsed time is 289s)
//
// Finally, when p < 1, a random number is chosen to decide whether
// the timeout should be heeded. This means the probability is
// used whether to trigger the timeout, so and timeouts are triggered
// probabilistically (non-deterministically). In the case of retries
// and distributed systems, this will spread them out them so a
// burdened remote server is not hit with
// simultaneous retries.

/*
func main() {
	to := 100 * time.Millisecond
	elapsed := 250 * time.Millisecond
	conf := ConfPROD
	transit := TransitRAM
	eval := SkepticalTimeout(to, elapsed, conf, transit)
	if eval {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}
}
*/
