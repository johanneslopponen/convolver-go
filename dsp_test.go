package main

import (
	"math/rand/v2"
	"testing"
)

func BenchmarkAdjustGain(b *testing.B) {
	audio := loadWavFile("input.wav")
	for i := 0; i < b.N; i++ {
		audio.AdjustGain(float32(rand.Float64()))
	}
}

func BenchmarkNormalizePeak(b *testing.B) {
	audio := loadWavFile("input.wav")
	for i := 0; i < b.N; i++ {
		audio.NormalizePeak(float32(rand.Float64()))
	}
}
