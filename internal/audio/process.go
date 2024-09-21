package audio

import "math"

type AudioProcessor struct {
	noiseThreshold 	float64
	sampleRate 		int
	numChannels 	int
}

func NewAudioProcessor(sampleRate, numChannels int) *AudioProcessor {
	return &AudioProcessor{
		noiseThreshold: 0.05, // moderate threshold
		sampleRate: sampleRate,
		numChannels: numChannels,
	}
}

func (ap *AudioProcessor) ReduceNoise(input []byte) []byte {
	output := make([]byte, len(input))
	samplesPerChannel := len(input) / (2 * ap.numChannels) // 16-bit samples

	for i := 0; i < samplesPerChannel; i++ {
		for ch := 0; ch < ap.numChannels; ch++ {
			idx := (i*ap.numChannels + ch) * 2
			sample := int16(input[idx]) | (int16(input[idx+1]) << 8)

			// simple noise gate
			if math.Abs(float64(sample)) < ap.noiseThreshold*32767 { // 32767 = max value for 16-bit audio
				sample = 0
			}

			// convert back to bytes
			output[idx] = byte(sample)
			output[idx+1] = byte(sample >> 8)
		}
	}
	return output
}

func (ap *AudioProcessor) SplitChannels(input []byte) ([]byte, []byte) {
	inputLength := len(input)
	leftChannel := make([]byte, inputLength/2)
	rightChannel := make([]byte, inputLength/2)

	for i := 0; i < inputLength; i += 4 {
		leftChannel[i/2] = input[i]
		leftChannel[i/2+1] = input[i+1]
		rightChannel[i/2] = input[i+2]
		rightChannel[i/2+1] = input[i+3]
	}
	return leftChannel, rightChannel
}