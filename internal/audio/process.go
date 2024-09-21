package audio

import "math"



type AudioProcessor struct {
	noiseThreshold 	float64
	attenuation		float64
	sampleRate 		int
	numChannels 	int
}

func NewAudioProcessor(sampleRate, numChannels int) *AudioProcessor {
	return &AudioProcessor{
		noiseThreshold: 0.05, // moderate threshold
		attenuation: 0.5, // reduce noise by 50%
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
				sample = int16(float64(sample) * ap.attenuation)
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
    leftLength := inputLength / 2
    if inputLength % 2 != 0 {
        leftLength = (inputLength + 1) / 2
    }
    leftChannel := make([]byte, leftLength)
    rightChannel := make([]byte, inputLength - leftLength)

    for i := 0; i < inputLength-1; i += 2 {
        leftChannel[i/2] = input[i]
        rightChannel[i/2] = input[i+1]
    }

    // Handle the last byte if input length is odd
    if inputLength % 2 != 0 {
        leftChannel[leftLength-1] = input[inputLength-1]
    }

    return leftChannel, rightChannel
}