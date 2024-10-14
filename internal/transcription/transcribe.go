package transcription

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"log"

	"github.com/alphacep/vosk-api/go"
)

type Transcriber struct {
	model 		*vosk.VoskModel
	recognizer 	*vosk.VoskRecognizer
	buffer 		strings.Builder
	lastWords	[]string
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

func (t *Transcriber) ProcessAudio(data []byte) ([]string, error) {
    result := t.recognizer.AcceptWaveform(data)
    var text string

    switch result {
    case 0:
        partialResult := t.recognizer.PartialResult()
        text = extractText(string(partialResult))
    case 1, 2:
        finalResult := t.recognizer.Result()
        text = extractText(string(finalResult))
    default:
        return nil, fmt.Errorf("unexpected result from AcceptWaveform: %d", result)
    }

    return t.getNewWords(text), nil
}

func (t *Transcriber) getNewWords(currentText string) []string {
    t.mu.Lock()
    defer t.mu.Unlock()

    currentWords := strings.Fields(currentText)
    newWords := []string{}

    for i := 0; i < len(currentWords); i++ {
        if i >= len(t.lastWords) || currentWords[i] != t.lastWords[i] {
            newWords = append(newWords, currentWords[i])
        }
    }

    t.lastWords = currentWords
    return newWords
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
		Text 		string `json:"text"`
		Partial 	string `json:"partial"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return ""
	}
	if data.Text != "" {
        return data.Text
    }
    return data.Partial
}

// func (t *Transcriber) processPartialResult(text string) string {
// 	t.mu.Lock()
// 	defer t.mu.Unlock()

// 	// compare with existing partial buffer
// 	existingWords := strings.Fields(t.partialBuffer.String())
// 	newWords := strings.Fields(text)

// 	var output strings.Builder
// 	for i, word := range newWords {
// 		if i >= len(existingWords) || word != existingWords[i] {
// 			output.WriteString(word + " ")
// 		}
// 	}

	// update partial buffer
	// t.partialBuffer.Reset()
	// t.partialBuffer.WriteString(text)

	// return output.String()
// }