package main

import "github.com/brettbuddin/fourier"

func (w *WavFile) AdjustGain(gain float32) {
	for ch := 0; ch < len(w.samples); ch++ {
		for i := range w.samples[ch] {
			w.samples[ch][i] *= gain
		}
	}
}

func (w *WavFile) NormalizePeak(peakValue float32) {
	var maxPeak float32
	for _, ch := range w.samples {
		for _, s := range ch {
			if s > maxPeak {
				maxPeak = s
			} else if -s > maxPeak {
				maxPeak = -s
			}
		}
	}
	if maxPeak == 0 {
		return
	}
	normalizationGain := (1 / maxPeak) * peakValue
	w.AdjustGain(normalizationGain)
}

func ConvolutionLength(audioLength, impulseLength int) int {
	return audioLength + impulseLength - 1
}

func toFloat64(in [][]float32) [][]float64 {
	out := make([][]float64, len(in))
	for ch, ch_samples := range in {
		out[ch] = make([]float64, len(ch_samples))
		for i, v := range ch_samples {
			out[ch][i] = float64(v)
		}
	}
	return out
}

func toFloat32(in [][]float64) [][]float32 {
	out := make([][]float32, len(in))
	for ch, ch_samples := range in {
		out[ch] = make([]float32, len(ch_samples))
		for i, v := range ch_samples {
			out[ch][i] = float32(v)
		}
	}
	return out
}

func FFTConvolve(audio, impulse *WavFile) *WavFile {
	assertSampleRate(audio, impulse)
	ir := toFloat64(impulse.samples)[0] // IR is always mono — use channel 0
	out := make([][]float64, len(audio.samples))
	for ch := 0; ch < len(audio.samples); ch++ {
		in := toFloat64(audio.samples)[ch]
		conv, _ := fourier.NewConvolver(4096, ir)
		out[ch] = make([]float64, len(in)+len(ir)-1)
		_ = conv.Convolve(out[ch], in, len(out[ch]))
	}
	return &WavFile{
		sampleRate: audio.sampleRate,
		channels:   len(out),
		samples:    toFloat32(out),
	}
}

func Blend(wet, dry *WavFile, blend float32) *WavFile {
	out := make([][]float32, len(wet.samples))
	for ch := 0; ch < len(wet.samples); ch++ {
		out[ch] = make([]float32, len(wet.samples[ch]))
		for i := range wet.samples[ch] {
			if i >= len(dry.samples[ch]) {
				out[ch][i] = wet.samples[ch][i]
			} else {
				out[ch][i] = wet.samples[ch][i]*blend + dry.samples[ch][i]*(1-blend)
			}
		}
	}
	return &WavFile{
		sampleRate: wet.sampleRate,
		channels:   len(out),
		samples:    out,
	}
}
