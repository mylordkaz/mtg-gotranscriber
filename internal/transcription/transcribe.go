package transcription

import (
	"encoding/json"
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

	recognizer.SetMaxAlternatives(0)
    recognizer.SetWords(1)

	return &Transcriber{
		model: model,
		recognizer: recognizer,
	}, nil
}

func (t *Transcriber) ProcessAudio(data []byte) (string, error) {
    result := t.recognizer.AcceptWaveform(data)
	switch result {
	case 0:
		partialResult := t.recognizer.PartialResult()
		text := extractText(string(partialResult))
		return text, nil
	case 1:
		finalResult := t.recognizer.Result()
        text := extractText(string(finalResult))
        t.appendToBuffer(text)
        return text, nil
	case 2:
		finalResult := t.recognizer.FinalResult()
        text := extractText(string(finalResult))
        t.appendToBuffer(text)
        return text, nil
	default:
		return "", fmt.Errorf("unexpected result from AcceptWaveform: %d", result)
	}
}
func (t *Transcriber) ResetBuffer() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buffer.Reset()
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
	var data struct {
		Text string `json:text`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return ""
	}
	return data.Text
}