package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
)

type WavFile struct {
	data        []byte
	sampleRate  int
	bitDepth    int
	channels    int
	duration    float64
	filename    string
	filesize    int
	audioFormat uint16      // 1=PCM, 3=IEEE float
	dataOffset  int         // byte offset of the audio data within file.data
	samples     [][]float32 // [channel][sample]
}

type AudioFile = WavFile
type ImpulseFile = WavFile

// findChunk scans RIFF chunks starting at offset 12 (after the RIFF/WAVE header)
// and returns the offset of the chunk's data payload and its size.
// Returns -1, 0 if not found.
func findChunk(data []byte, id string) (dataOffset int, size int) {
	pos := 12
	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		payloadOffset := pos + 8
		if chunkID == id {
			return payloadOffset, chunkSize
		}
		// Chunks are padded to even byte boundaries
		step := chunkSize
		if step%2 != 0 {
			step++
		}
		pos = payloadOffset + step
	}
	return -1, 0
}

// Load a wave file and create an object of type WavFile
// See https://en.wikipedia.org/wiki/WAV#WAV_file_header
func loadWavFile(filename string) *WavFile {
	file := WavFile{filename: filename}
	data, err := os.ReadFile(file.filename)
	file.data = data

	if err != nil {
		log.Fatal(err)
	}

	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		log.Fatal("loadWavFile: not a valid RIFF/WAVE file")
	}

	// Locate fmt chunk (may be 16, 18, or 40 bytes)
	fmtOffset, fmtSize := findChunk(data, "fmt ")
	if fmtOffset < 0 || fmtSize < 16 {
		log.Fatal("loadWavFile: fmt chunk not found or too small")
	}
	fmt_ := data[fmtOffset : fmtOffset+fmtSize]

	file.audioFormat = binary.LittleEndian.Uint16(fmt_[0:2])
	file.channels = int(binary.LittleEndian.Uint16(fmt_[2:4]))
	file.sampleRate = int(binary.LittleEndian.Uint32(fmt_[4:8]))
	file.bitDepth = int(binary.LittleEndian.Uint16(fmt_[14:16]))

	// Locate data chunk
	dataChunkOffset, dataChunkSize := findChunk(data, "data")
	if dataChunkOffset < 0 {
		log.Fatal("loadWavFile: data chunk not found")
	}
	file.dataOffset = dataChunkOffset
	file.filesize = dataChunkSize

	// duration
	file.duration = float64(dataChunkSize) / (float64(file.bitDepth) / 8 * float64(file.sampleRate) * float64(file.channels))

	// extract samples to struct
	file.samples = file.ToFloat32PerChannel()

	return &file
}

func assertSampleRate(audio, impulse *WavFile) {
	if audio.sampleRate != impulse.sampleRate {
		log.Fatal("Impulse responce does not have same sample rate as input audio file.")
	}
}

func (w *WavFile) ToFloat32PerChannel() [][]float32 {
	// Use the discovered data chunk offset
	data := w.data[w.dataOffset : w.dataOffset+w.filesize]
	bytesPerSample := w.bitDepth / 8
	frameSize := bytesPerSample * w.channels

	if frameSize == 0 {
		log.Fatal("ToFloat32PerChannel: ogiltig frameSize (bitDepth/kanaler)")
	}

	framesAmount := len(data) / frameSize

	out := make([][]float32, w.channels)
	for ch := 0; ch < w.channels; ch++ {
		out[ch] = make([]float32, framesAmount)
	}

	switch w.bitDepth {
	case 16:
		// int16 PCM → float32 [-1,1]
		const norm = 32768.0
		for n := 0; n < framesAmount; n++ {
			frameOffset := n * frameSize
			for ch := 0; ch < w.channels; ch++ {
				off := frameOffset + ch*bytesPerSample
				if off+1 >= len(data) {
					break
				}
				v := int16(binary.LittleEndian.Uint16(data[off : off+2]))
				out[ch][n] = float32(v) / norm
			}
		}

	case 24:
		// int24 PCM → float32 [-1,1]
		const norm = float32(1 << 23) // 2^23
		for n := 0; n < framesAmount; n++ {
			frameOffset := n * frameSize
			for ch := 0; ch < w.channels; ch++ {
				off := frameOffset + ch*bytesPerSample
				if off+2 >= len(data) {
					break
				}

				b0 := uint32(data[off])
				b1 := uint32(data[off+1])
				b2 := uint32(data[off+2])

				v := int32(b0 | (b1 << 8) | (b2 << 16))
				// sign-extend 24 bit → 32 bit
				if v&0x00800000 != 0 {
					v |= ^0x00ffffff
				}

				out[ch][n] = float32(v) / norm
			}
		}

	case 32:
		if w.audioFormat == 3 {
			// IEEE 754 float32 — values may legitimately exceed [-1, 1]
			for n := 0; n < framesAmount; n++ {
				frameOffset := n * frameSize
				for ch := 0; ch < w.channels; ch++ {
					off := frameOffset + ch*bytesPerSample
					if off+3 >= len(data) {
						break
					}

					bits := binary.LittleEndian.Uint32(data[off : off+4])
					out[ch][n] = math.Float32frombits(bits)
				}
			}
		} else if w.audioFormat == 1 {
			// int32 PCM → float32 [-1, 1]
			const norm = float32(1 << 31) // 2^31 = 2147483648
			for n := 0; n < framesAmount; n++ {
				frameOffset := n * frameSize
				for ch := 0; ch < w.channels; ch++ {
					off := frameOffset + ch*bytesPerSample
					if off+3 >= len(data) {
						break
					}

					v := int32(binary.LittleEndian.Uint32(data[off : off+4]))
					out[ch][n] = float32(v) / norm
				}
			}
		} else {
			log.Fatalf("ToFloat32PerChannel: 32-bit with AudioFormat=%d is not supported (expected 1 for int32 PCM or 3 for IEEE float)", w.audioFormat)
		}

	default:
		log.Fatalf("ToFloat32PerChannel: bit depth %d stöds inte", w.bitDepth)
	}

	return out
}

func (w *WavFile) SaveAsFloat32Wav(filename string) error {
	const headerSize = 44

	if len(w.samples) == 0 {
		return fmt.Errorf("SaveAsFloat32Wav: no samples")
	}

	channelsAmount := len(w.samples)
	frames := len(w.samples[0])

	// check that all channels have the same length as ch0
	for ch := 1; ch < channelsAmount; ch++ {
		if len(w.samples[ch]) != frames {
			return fmt.Errorf("SaveAsFloat32Wav: channels are of differing lengths")
		}
	}

	bytesPerSample := 4 // float32
	byteRate := w.sampleRate * channelsAmount * bytesPerSample
	blockAlign := channelsAmount * bytesPerSample
	dataSize := frames * channelsAmount * bytesPerSample
	chunkSize := 36 + dataSize

	buf := make([]byte, headerSize+dataSize)

	// Build wav file header
	// RIFF header
	copy(buf[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(chunkSize))
	copy(buf[8:12], []byte("WAVE"))
	// fmt chunk
	copy(buf[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(buf[16:20], 16)                     // Subchunk1Size
	binary.LittleEndian.PutUint16(buf[20:22], 3)                      // AudioFormat = 3 (IEEE float)
	binary.LittleEndian.PutUint16(buf[22:24], uint16(channelsAmount)) // NumChannels
	binary.LittleEndian.PutUint32(buf[24:28], uint32(w.sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(buf[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(buf[34:36], 32)

	// Write data
	// data chunk
	copy(buf[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))

	// interleave and write float32-samples
	data := buf[headerSize:]
	idx := 0
	for n := 0; n < frames; n++ {
		for ch := 0; ch < channelsAmount; ch++ {
			s := w.samples[ch][n]
			bits := math.Float32bits(s)
			binary.LittleEndian.PutUint32(data[idx:idx+4], bits)
			idx += 4
		}
	}
	return os.WriteFile(filename, buf, 0644)
}
