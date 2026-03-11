# reverb_convolver

A real-time audio convolution reverb processor written in Go. It captures live audio from your microphone, convolves it with an impulse response (IR) file using FFT-based partitioned convolution, and outputs the result to your speakers in real time.

Convolution reverb works by mathematically convolving a dry audio signal with a recorded impulse response of a physical space (e.g. a church), producing reverb that simulates the acoustics of that space.

The project allso includes functions to convolve wav files instead of doing realtime convolution.
## Prerequisites

- Go 1.25+
- PortAudio C library installed on your system
- An audio input device (microphone) and output device (speakers/headphones)
- An impulse response WAV file named `kyrka.wav` in the working directory

### Installing PortAudio

On Debian/Ubuntu:
```bash
sudo apt install portaudio19-dev
```

On macOS:
```bash
brew install portaudio
```

## Build

```bash
go build -o reverb_convolver
```

## Usage

```bash
./reverb_convolver [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-sr` | `0` | Target sample rate. `0` uses the device default. |
| `-buffSize` | `0` | Buffer size. `0` derives from device default latency. |

### Example

```bash
./reverb_convolver -sr 48000 -buffSize 1024
```

The program streams audio until you press `Ctrl+C` to stop.

## Project Structure

| File | Description |
|------|-------------|
| `convolver.go` | Main entry point, CLI flags, signal handling |
| `realtime_io.go` | PortAudio stream setup and real-time I/O |
| `realtime_dsp.go` | Real-time DSP engine (per-channel FFT convolution, gain, wet/dry blend) |
| `dsp.go` | Offline DSP operations (full-file convolution, normalization, mixing) |
| `wav_io.go` | WAV file reading/writing (16/24/32-bit PCM and 32-bit float) |

## Dependencies

- [fourier](https://github.com/brettbuddin/fourier) -- FFT-based partitioned convolution
- [portaudio](https://github.com/gordonklaus/portaudio) -- Go bindings for PortAudio

## Running Benchmarks

```bash
go test -bench . -benchmem
```
