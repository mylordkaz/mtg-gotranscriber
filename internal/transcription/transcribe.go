package transcription

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
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
    fmt.Printf("Debug: Processing chunk of size %d bytes\n", len(chunk))

    // Reset file for new write
    t.tempFile.Seek(0, 0)
    t.tempFile.Truncate(0)

    _, err := t.tempFile.Write(chunk)
    if err != nil {
        return "", fmt.Errorf("error writing to temp file: %v", err)
    }

    // Ensure all data is written to disk
    t.tempFile.Sync()

    fmt.Printf("Debug: Wrote %d bytes to temp file %s\n", len(chunk), t.tempFile.Name())

    scriptPath := "/Users/MyLord/goProject/mtg-gotranscriber/scripts/whisper_transcriber.py"
    cmd := exec.Command("python3", scriptPath, t.tempFile.Name())
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

    output, err := cmd.CombinedOutput()
    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            if exitError.ExitCode() == -1 { // interrupted
                return "", fmt.Errorf("transcription interrupted")
            }
        }
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