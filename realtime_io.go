package main

import (
	"fmt"
	"time"

	"github.com/gordonklaus/portaudio"
)

type RealtimeIO struct {
	Stream          *portaudio.Stream
	SampleRate      float64
	In              [][]float32
	Out             [][]float32
	FramesPerBuffer int
	ProcessAudio    func(in, out [][]float32) // user-defined DSP callback
}

// Create IO Audio stream using a PortAudio callback for reliable timing.
// The processFunc is called by PortAudio whenever audio data is needed.
func InitRealtimeIO(targetSampleRate float64, bufferSize int, processFunc func(in, out [][]float32)) (*RealtimeIO, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	h, err := portaudio.DefaultHostApi()
	if err != nil {
		portaudio.Terminate()
		return nil, err
	}

	inDev := h.DefaultInputDevice
	outDev := h.DefaultOutputDevice

	if inDev == nil {
		portaudio.Terminate()
		return nil, fmt.Errorf("no default input device found")
	}
	if outDev == nil {
		portaudio.Terminate()
		return nil, fmt.Errorf("no default output device found")
	}

	params := portaudio.LowLatencyParameters(inDev, outDev)
	if targetSampleRate > 0 {
		params.SampleRate = targetSampleRate
	}
	sampleRate := params.SampleRate

	// Derive framesPerBuffer from the chosen latency and sample rate.
	outputLatency := params.Output.Latency
	if outputLatency <= 0 {
		outputLatency = 10 * time.Millisecond
	}
	framesPerBuffer := int(sampleRate * outputLatency.Seconds())
	if framesPerBuffer < 64 {
		framesPerBuffer = 64
	}
	if bufferSize > 0 {
		framesPerBuffer = bufferSize / 2
	}
	params.FramesPerBuffer = framesPerBuffer

	inCh := params.Input.Channels
	outCh := params.Output.Channels

	rio := &RealtimeIO{
		SampleRate:      sampleRate,
		FramesPerBuffer: framesPerBuffer,
		ProcessAudio:    processFunc,
	}

	// Pre-allocate channel slices (the inner slices are set by PortAudio per callback)
	rio.In = make([][]float32, inCh)
	rio.Out = make([][]float32, outCh)

	// Open a callback-based stream. PortAudio calls this function from its
	// audio thread whenever it needs samples, which avoids the timing issues
	// inherent in the blocking Read()/Write() approach.
	stream, err := portaudio.OpenStream(params, func(in, out [][]float32) {
		rio.In = in
		rio.Out = out
		if rio.ProcessAudio != nil {
			rio.ProcessAudio(in, out)
		}
	})
	if err != nil {
		portaudio.Terminate()
		return nil, err
	}
	rio.Stream = stream

	fmt.Printf("Sample Rate: %.0f Hz\n", sampleRate)
	fmt.Printf("Channels: In %d, Out %d\n", inCh, outCh)
	fmt.Printf("Frames Per Buffer: %d\n", framesPerBuffer)

	return rio, nil
}
