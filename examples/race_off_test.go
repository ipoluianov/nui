//go:build !race

package examples

// raceEnabled is true under the race detector, which makes the monkey
// test about ten times slower
const raceEnabled = false
