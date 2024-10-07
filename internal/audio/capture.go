package audio

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
        "-i", ":BlackHole 2ch",
        "-acodec", "pcm_s16le",
        "-ar", "16000", // output sample rate to 16kHz
        "-ac", "1",  // set to mono
        "-f", "s16le",
		"-af", "dynaudnorm=f=200:g=3:p=0.91,highpass=f=80,lowpass=f=7500",
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
	return ac.cmd.Process.Kill()
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