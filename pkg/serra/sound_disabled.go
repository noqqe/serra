//go:build !darwin && !windows && !cgo

package serra

// No audio backend available in this build (see sound.go), so the cues are
// no-ops. They are called as detached goroutines and nothing reads a result,
// so dropping them changes nothing but the noise.

func playSoundPositive() {}

func playSoundNegative() {}

func playSoundCash() {}
