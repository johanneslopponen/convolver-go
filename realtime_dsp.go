package main

import "github.com/brettbuddin/fourier"

type DSP struct {
	impulse   []float64
	convolver []*fourier.Convolver
	gain      float32
	blend     float32
}

func (d *DSP) Process(in, out [][]float32) {
	RelayAudio(in, out)
	// Apply convolution to the output buffer in-place
	d.ConvolveRT(out)
	d.GainRT(out, d.gain)
	BlendRT(out, in, d.blend)
}

func RelayAudio(in, out [][]float32) {
	inChannels := len(in)
	outChannels := len(out)

	if inChannels == 1 && outChannels > 1 {
		// Mono input to multi-channel output: duplicate signal
		for ch := 0; ch < outChannels; ch++ {
			copy(out[ch], in[0])
		}
	} else {
		// Standard mapping: copy channel for channel
		channels := inChannels
		if outChannels < channels {
			channels = outChannels
		}
		for ch := 0; ch < channels; ch++ {
			copy(out[ch], in[ch])
		}
	}
}
func (d *DSP) loadImpulse64(ir *WavFile) {
	// We only take the first channel of the impulse response
	d.impulse = toFloat64(ir.samples)[0]
}

func (d *DSP) Init(numChannels int) {
	d.convolver = make([]*fourier.Convolver, numChannels)
	for ch := 0; ch < numChannels; ch++ {
		// NewConvolver(4096) is a good balance for quality/performance
		d.convolver[ch], _ = fourier.NewConvolver(4096, d.impulse)
	}
}

func (d *DSP) ConvolveRT(buf [][]float32) {
	input := toFloat64(buf)
	for ch := 0; ch < len(buf); ch++ {
		if ch >= len(d.convolver) {
			break
		}
		// Convolve into a temporary float64 slice
		out64 := make([]float64, len(input[ch]))
		_ = d.convolver[ch].Convolve(out64, input[ch], len(input[ch]))

		// Copy back to output float32 buffer
		for i, v := range out64 {
			buf[ch][i] = float32(v)
		}
	}
}

func (d *DSP) GainRT(in [][]float32, gain float32) [][]float32 {
	for ch := 0; ch < len(in); ch++ {
		for i := range in[ch] {
			in[ch][i] = in[ch][i] * gain
		}
	}
	return in
}

func BlendRT(in [][]float32, og [][]float32, blend float32) [][]float32 {
	inChannels := len(in)
	ogChannels := len(og)

	for ch := 0; ch < inChannels; ch++ {
		// Determine which original channel to blend with
		ogCh := ch
		if ogChannels == 1 {
			ogCh = 0
		} else if ch >= ogChannels {
			// If we have more output channels than input channels,
			// just stop blending or handle as needed.
			// In our RelayAudio mono->stereo case, og is 'in' (mono).
			break
		}

		for i := range in[ch] {
			in[ch][i] = in[ch][i]*blend + og[ogCh][i]*(1-blend)
		}
	}
	return in
}
