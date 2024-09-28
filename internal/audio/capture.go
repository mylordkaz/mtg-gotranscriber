package audio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)


type AudioCapture struct {
	cmd *exec.Cmd
	stdout io.ReadCloser
	reader *bufio.Reader
}
func NewCaptureAudio() (*AudioCapture, error) {
	cmd := exec.Command("ffmpeg",
        "-f", "avfoundation",
        "-i", ":0",
        "-acodec", "pcm_s16le",
        "-ar", "44100",
        "-ac", "2",
        "-f", "s16le",
        "-")
	
		cmd.Stderr = os.Stderr

		stdout, err := cmd.StdoutPipe()
    	if err != nil {
        return nil, fmt.Errorf("error creating StdoutPipe for FFmpeg: %v", err)
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
	if ac.cmd != nil && ac.cmd.Process != nil {
        fmt.Println("Debug: Stopping ffmpeg process")
        
		if stdin, err := ac.cmd.StdinPipe(); err == nil {
			stdin.Close()
		}

		done := make(chan error, 1)
		go func() {
			done <- ac.cmd.Wait()
		}()

		select {
		case err := <-done:
			return err
		case <- time.After(5 * time.Second):
			return ac.cmd.Process.Kill()
		}
    }
    return nil
}
func (ac *AudioCapture) ReadChunk(bufferSize int) ([]byte, error) {
	buffer := make([]byte, bufferSize)
    n, err := io.ReadFull(ac.reader, buffer)

    if err != nil {
        if err == io.EOF {
            return nil, fmt.Errorf("EOF reached, no more audio data available")
        }
        if err == io.ErrUnexpectedEOF {
            return buffer[:n], nil  // Return partial buffer
        }
        return nil, fmt.Errorf("error reading audio data: %v", err)
    }
    return buffer, nil
}