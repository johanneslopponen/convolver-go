package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gordonklaus/portaudio"
)

// "github.com/brettbuddin/fourier"

func main() {
	/* audioPtr := flag.String("audio", "input.wav", "Input audio wav file")
	impulsePtr := flag.String("impulse", "impulse.wav", "Impulse responce wav file")
	flag.Parse()
	audio := loadWavFile(*audioPtr)
	impulse := loadWavFile(*impulsePtr)
	convolved := FFTConvolve(audio, impulse)
	convolved.NormalizePeak(0.9)
	convolved.SaveAsFloat32Wav("convolved.wav")
	blended := Blend(convolved, audio, 0.5)
	blended.SaveAsFloat32Wav("blended.wav") */
	sr := flag.Float64("sr", 0, "Target sample rate (0 for device default)")
	buffSize := flag.Int("buffSize", 0, "Buffer size (0 for device default)")
	flag.Parse()

	ir := loadWavFile("kyrka.wav")
	d := &DSP{}
	d.loadImpulse64(ir)
	d.gain = 0.15
	d.blend = 1
	io, err := InitRealtimeIO(*sr, *buffSize, d.Process)
	if err != nil {
		fmt.Printf("Error initializing realtime IO: %v\n", err)
		return
	}
	// Correctly initialize one convolver for each output channel
	d.Init(len(io.Out))
	defer portaudio.Terminate()
	defer io.Stream.Close()

	if err := io.Stream.Start(); err != nil {
		fmt.Printf("Error starting stream: %v\n", err)
		return
	}
	defer io.Stream.Stop()

	fmt.Printf("Streaming audio... Press Ctrl+C to stop.\n")

	// Set up channel to listen for interrupt signal (Ctrl+C)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Block until interrupt
	<-sig
	fmt.Println("\nStopping audio stream...")
}

// Todo:
// 1. Load audio and impulse responce
// 2. Check sample rates
// 3. Sum to mono (Eventually handle stereo with multithreading)
// 4. Perform the convolution using FFT
// 5. Normalize peak levels
// 6. Blend with dry signal
// 7. Write to file

// New features:
// Create a realtime engine with connections to portaudio for realtime reverb
// Implement custom FFT algorithm
// TUI interface

// https://github.com/avelino/awesome-go/blob/main/README.md#audio-and-music
