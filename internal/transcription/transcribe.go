package transcription

import (
    "fmt"
    "os/exec"
    "strings"
    "sync"
    "os"
)

type Transcriber struct {
    buffer strings.Builder
    mu     sync.Mutex
    tempFile *os.File
}

func NewTranscriber() (*Transcriber, error) {
    tempFile, err := os.CreateTemp("", "temp_audio_*.wav")
    if err != nil {
        return nil, fmt.Errorf("error creating temporary file: %v", err)
    }
    return &Transcriber{tempFile: tempFile}, nil
}

func (t *Transcriber) ProcessAudioChunk(chunk []byte) (string, error) {
    _, err := t.tempFile.Write(chunk)
    if err != nil {
        return "", fmt.Errorf("error writing to temp file: %v", err)
    }

	scriptPath := "/Users/MyLord/goProject/mtg-gotranscriber/scripts/whisper_transcriber.py"
    cmd := exec.Command("python3", scriptPath, t.tempFile.Name())
    output, err := cmd.CombinedOutput()
    if err != nil {
		return "", fmt.Errorf("error during transcription: %v\nOutput: %s", err, string(output))
    }
    transcript := strings.TrimSpace(string(output))
    t.appendToBuffer(transcript)

    return transcript, nil
}

func (t *Transcriber) Finalize() string {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.tempFile.Close()
    os.Remove(t.tempFile.Name())
    return t.buffer.String()
}

func (t *Transcriber) appendToBuffer(text string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.buffer.WriteString(text + " ")
}