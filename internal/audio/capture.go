package audio

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)


type AudioCapture struct {
	cmd *exec.Cmd
	stdout io.ReadCloser
	reader *bufio.Reader
}
func NewCaptureAudio() (*AudioCapture, error) {
	cmd := exec.Command("ffmpeg",
	"-f", "avfoundation", 
	"-i", ":0", // Capture system audio
	"-acodec", "pcm_s16le",
	"-ar", "44100",
	"-ac", "2",
	"-f", "s16le",
	"_")

	stdout, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating StdoutPipe for ffmpeg: %v", err)
	}

	return &AudioCapture{
		cmd: cmd,
		stdout: stdout,
		reader: bufio.NewReader(stdout),
	}, nil
}

func (ac *AudioCapture) Start() error {
	return ac.cmd.Start()
}
func (ac *AudioCapture) Stop() error {
	return ac.cmd.Process.Kill()
}
func (ac *AudioCapture) ReadChunk(bufferSize int) ([]byte, error) {
	buffer := make([]byte, bufferSize)
	n, err := io.ReadFull(ac.reader, buffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buffer[:n], nil
}