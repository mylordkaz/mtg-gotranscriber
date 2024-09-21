package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mylordkaz/mtg-gotranscriber/internal/audio"
)

func main() {
	// initialize audio capture
	capture, err := audio.NewCaptureAudio()
	if err != nil {
		fmt.Println("Error creating audio capture:", err)
		return
	}

	processor := audio.NewAudioProcessor(44100, 2) // 44.1khz stereo audio

	err = capture.Start()
	if err != nil {
		fmt.Println("Error starting audio capture:", err)
		return
	}
	fmt.Println("Audio capture started. Press Ctrl+C to stop.")

	// signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			chunk, err := capture.ReadChunk(4096)
			if err != nil {
				fmt.Println("Error reading audio chunk:", err)
				return
			}
			processedChunk := processor.ReduceNoise(chunk)
			leftChan, rightChan := processor.SplitChannels(processedChunk)

			fmt.Printf("Processed chunk: Let channel: %d bytes, Right channel: %d bytes\n", len(leftChan), len(rightChan))
		}
	}()

	// wait for termination signal
	<-sigChan

	fmt.Println("\nStopping audio capture...")
	err = capture.Stop()
	if err != nil {
		fmt.Println("Error stopping audio capture:", err)
	}

	// initialize speech recognition
	// TODO: implement speech recognition

	// transcription loop
		// capture audio
		// process audio
		// get transcription
			// use goroutine to handle transcription
		// output transcription, print to the console
		// append each transcription chunk to a buffer
		// when session ends, write entire buffer to a file
}