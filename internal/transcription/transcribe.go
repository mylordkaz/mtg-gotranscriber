package transcription

import (
	"fmt"
	"strings"
	"sync"

	"github.com/alphacep/vosk-api/go"
)




type Transcriber struct {
	model 		*vosk.VoskModel
	recognizer 	*vosk.VoskRecognizer
	buffer 		strings.Builder
	mu 			sync.Mutex
}

func NewTranscriber(modelPath string, sampleRate float64) (*Transcriber, error) {
	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %v", err)
	}

	recognizer, err := vosk.NewRecognizer(model, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("failed to create recognizer: %v", err)
	}

	return &Transcriber{
		model: model,
		recognizer: recognizer,
	}, nil
}

func (t *Transcriber) ProcessAudio(data []byte) string {
    result := t.recognizer.AcceptWaveform(data)
	switch result {
	case 0:
		return ""
	case 1:
		partialResult := t.recognizer.PartialResult()
		text := extractText(string(partialResult))
		return text
	case 2:
		finalResult := t.recognizer.FinalResult()
		text := extractText(string(finalResult))
		return text
	default:
		return ""
	}
}

// call after finished audio capture to ensure we got all possible transcription.
func (t *Transcriber) Finalize() string {
	result := t.recognizer.FinalResult()
	text := extractText(string(result))
	t.appendToBuffer(text)
	return text
}

func (t *Transcriber) Close() {
	t.recognizer.Free()
	t.model.Free()
}

func (t *Transcriber) GetFullTranscription() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buffer.String()
}

func (t *Transcriber) appendToBuffer(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buffer.WriteString(text + " ")
}

func extractText(result string) string {
	parts := strings.Split(result, "\"text\" : \"")
	if len(parts) < 2 {
		return ""
	}
	return strings.Trim(parts[1], "\"\n}")
}