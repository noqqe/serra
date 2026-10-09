// Audio cues need an audio backend. oto talks to CoreAudio via purego on
// darwin and to WinMM/WASAPI via syscall on windows, so those build without
// cgo - everywhere else (linux, the BSDs) it binds ALSA through cgo. Release
// builds are CGO_ENABLED=0, so on those platforms the cues are compiled out
// entirely by sound_disabled.go instead of failing to link.
//go:build darwin || windows || cgo

package serra

import (
	"bytes"
	"embed"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

// The cues are compiled into the binary so a release is a single file and
// does not care about the working directory it is started from.
//
//go:embed sounds/*.mp3
var soundFiles embed.FS

var once sync.Once
var ctx *oto.Context

func initSound() {

	// Prepare an Oto context (this will use your default audio device) that will
	// play all our sounds. Its configuration can't be changed later.
	op := &oto.NewContextOptions{}

	// Usually 44100 or 48000. Other values might cause distortions in Oto
	op.SampleRate = 44100

	// Number of channels (aka locations) to play sounds from. Either 1 or 2.
	// 1 is mono sound, and 2 is stereo (most speakers are stereo).
	op.ChannelCount = 2

	// Format of the source. go-mp3's format is signed 16bit integers.
	op.Format = oto.FormatSignedInt16LE

	// Remember that you should **not** create more than one context
	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		// No audio device (headless server, container, CI). Leave ctx nil
		// and let playSound() skip the cue instead of killing the command.
		Logger().Debugf("oto.NewContext failed: %v", err)
		return
	}
	// It might take a bit for the hardware audio devices to be ready, so we wait on the channel.
	<-readyChan

	ctx = otoCtx
}

func playSound(file string) {

	// These run as fire-and-forget goroutines, so a broken cue must never
	// take the whole command down with it.
	fileBytes, err := soundFiles.ReadFile("sounds/" + file)
	if err != nil {
		Logger().Debugf("reading mp3 failed: %v", err)
		return
	}

	// Convert the pure bytes into a reader object that can be used with the mp3 decoder
	fileBytesReader := bytes.NewReader(fileBytes)

	// Decode file
	decodedMp3, err := mp3.NewDecoder(fileBytesReader)
	if err != nil {
		Logger().Debugf("mp3.NewDecoder failed: %v", err)
		return
	}

	once.Do(initSound)
	if ctx == nil {
		return
	}

	// Create a new 'player' that will handle our sound. Paused by default.
	player := ctx.NewPlayer(decodedMp3)

	// Play starts playing the sound and returns without waiting for it (Play() is async).
	player.Play()

	// We can wait for the sound to finish playing using something like this
	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}

}

func playSoundPositive() {
	playSound("success.mp3")
}

func playSoundNegative() {
	playSound("error.mp3")
}

func playSoundCash() {
	playSound("cash.mp3")
}
