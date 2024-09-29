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
	stdin  io.WriteCloser
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
		"-y",
        "-")
	
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
        return nil, fmt.Errorf("error creating StdoutPipe for FFmpeg: %v", err)
	}
	stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, fmt.Errorf("error creating StdinPipe for FFmpeg: %v", err)
    }
	

	return &AudioCapture{
		cmd: cmd,
		stdout: stdout,
		stdin: stdin,
		reader: bufio.NewReader(stdout),
	}, nil
}

func (ac *AudioCapture) Start() error {
	return ac.cmd.Start()
}

func (ac *AudioCapture) Stop() error {
	if ac.cmd != nil && ac.cmd.Process != nil {
        // Close stdin to signal FFmpeg to stop
        ac.stdin.Close()

        // Wait for FFmpeg to exit with a timeout
        done := make(chan error, 1)
        go func() {
            done <- ac.cmd.Wait()
        }()

        select {
        case err := <-done:
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 255 {
				return nil
			}
            return err
        case <-time.After(5 * time.Second):
            // Force kill if it doesn't exit within 5 seconds
            ac.cmd.Process.Kill()
            return fmt.Errorf("FFmpeg did not exit within timeout period")
        }
    }
    return nil
}

func (ac *AudioCapture) ReadChunk(bufferSize int) ([]byte, error) {
	buffer := make([]byte, bufferSize)
    n, err := io.ReadFull(ac.reader, buffer)

    if err != nil {
        if err == io.EOF {
            return nil, io.EOF
        }
        if err == io.ErrUnexpectedEOF {
            return buffer[:n], nil  // Return partial buffer
        }
        return nil, fmt.Errorf("error reading audio data: %v", err)
    }
    return buffer, nil
}