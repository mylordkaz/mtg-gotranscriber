package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"sync"
	"syscall"

	"github.com/mylordkaz/mtg-gotranscriber/internal/audio"
	
)

func main() {
    os.Stdout = os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	// initialize audio capture
	capture, err := audio.NewCaptureAudio()
	if err != nil {
		fmt.Println("Error creating audio capture:", err)
		return
	}

	processor := audio.NewAudioProcessor(16000, 1) // 16kHz mono

	// initialize transcriber
	modelPath := filepath.Join("internal", "transcription", "models", "ja-model")
	transcriber, err := transcription.NewTranscriber(modelPath, 16000)
	if err != nil {
		fmt.Println("Error creating transcriber:", err)
		return
	}
	defer transcriber.Close()

    // start audio capture
	err = capture.Start()
	if err != nil {
		fmt.Println("Error starting audio capture:", err)
		return
	}
	fmt.Println("Audio capture started. Press Ctrl+C to stop.")

	// signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // create WAV file
	outputFile, err := os.Create("output.wav")
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	writeWAVHeader(outputFile, 16000, 1, 16)

    done := make(chan struct{})
    var wg sync.WaitGroup
	var mu sync.Mutex
	totalBytesWritten := 0

    wg.Add(1)
	go processAudio(&wg, capture, processor, transcriber, outputFile, &mu, &totalBytesWritten, done)

    // Wait for termination signal
    <-sigChan

    fmt.Println("\nStopping audio capture...")
    if err := capture.Stop(); err != nil {
        log.Printf("Error stopping audio capture: %v", err)
    }

    close(done)

    // Wait for audio processing to complete
    wg.Wait()

    // Finalize transcription and save to file
    finalizeTranscription(transcriber)

    // Update WAV header
    mu.Lock()
    if err := updateWAVHeader(outputFile, totalBytesWritten); err != nil {
        log.Printf("Error updating WAV header: %v", err)
    }
    mu.Unlock()
}

func processAudio(wg *sync.WaitGroup, capture *audio.AudioCapture, processor *audio.AudioProcessor, transcriber *transcription.Transcriber, outputFile *os.File, mu *sync.Mutex, totalBytesWritten *int, done chan struct{}) {
    defer wg.Done()

    for {
        select {
        case <-done:
            return
        default:
            chunk, err := capture.ReadChunk(512)
            if err != nil {
                if err == io.EOF {
                    log.Println("End of audio stream reached")
                    return
                }
                log.Printf("Error reading audio chunk: %v", err)
                continue
            }

            // Process audio chunk (noise reduction currently not in use)
            processedChunk := processor.ReduceNoise(chunk)
            if len(processedChunk) == 0 {
                continue
            }

            // Write processed audio to file
            n, err := outputFile.Write(chunk)
            if err != nil {
                log.Printf("Error writing to file: %v", err)
                return
            }

            mu.Lock()
            *totalBytesWritten += n
            mu.Unlock()

            // Process audio for transcription
            transcript, err := transcriber.ProcessAudio(chunk)
            if err != nil {
                log.Printf("Error processing audio for transcription: %v", err)
                continue
            }
            if transcript != "" {
                fmt.Printf("Transcription: %s\n", transcript)
                os.Stdout.Sync() // force flush
            }
        }
    }

}

func finalizeTranscription(transcriber *transcription.Transcriber) {
    finalTranscription := transcriber.Finalize()
    if finalTranscription != "" {
        fmt.Printf("Final transcription: %s\n", finalTranscription)
    }

    fullTranscription := transcriber.GetFullTranscription()
    if err := os.WriteFile("transcription.txt", []byte(fullTranscription), 0644); err != nil {
        log.Printf("Error saving transcription to file: %v", err)
    }
}

func writeWAVHeader(file *os.File, sampleRate, numChannels, bitsPerSample int) {
    // RIFF chunk
    file.WriteString("RIFF")
    file.Write([]byte{0, 0, 0, 0}) // File size, to be filled later
    file.WriteString("WAVE")

    // fmt sub-chunk
    file.WriteString("fmt ")
    binary.Write(file, binary.LittleEndian, int32(16)) // Chunk size
    binary.Write(file, binary.LittleEndian, int16(1)) // Audio format (PCM)
    binary.Write(file, binary.LittleEndian, int16(numChannels))
    binary.Write(file, binary.LittleEndian, int32(sampleRate))
    binary.Write(file, binary.LittleEndian, int32(sampleRate*numChannels*bitsPerSample/8)) // Byte rate
    binary.Write(file, binary.LittleEndian, int16(numChannels*bitsPerSample/8)) // Block align
    binary.Write(file, binary.LittleEndian, int16(bitsPerSample))

    // data sub-chunk
    file.WriteString("data")
    file.Write([]byte{0, 0, 0, 0}) // Data size, to be filled later
}
func updateWAVHeader(file *os.File, dataSize int) error {
    // Update RIFF chunk size
    file.Seek(4, 0)
    binary.Write(file, binary.LittleEndian, int32(dataSize+36))

    // Update data sub-chunk size
    file.Seek(40, 0)
    binary.Write(file, binary.LittleEndian, int32(dataSize))

    return nil
}