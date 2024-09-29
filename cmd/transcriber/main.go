package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mylordkaz/mtg-gotranscriber/internal/audio"
	"github.com/mylordkaz/mtg-gotranscriber/internal/transcription"
)

func main() {
    // Initialize audio capture
    capture, err := audio.NewCaptureAudio()
    if err != nil {
        fmt.Println("Error creating audio capture:", err)
        return
    }

    // initialze transcription
    transcriber, err := transcription.NewTranscriber()
    if err != nil {
        fmt.Println("Error creating transcriber:", err )
        return
    }


    processor := audio.NewAudioProcessor(44100, 2) // 44.1khz stereo audio

    err = capture.Start()
    if err != nil {
        fmt.Println("Error starting audio capture:", err)
        return
    }
    fmt.Println("Audio capture started. Press Ctrl+C to stop.")

    // Signal handling for graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    outputFile, err := os.Create("output.wav")
    if err != nil {
        fmt.Println("Error creating output file:", err)
        return
    }
    defer outputFile.Close()

    writeWAVHeader(outputFile, 44100, 2, 16)

    var mu sync.Mutex
    totalBytesWritten := 0

    // ensure all go routines finish before exit
    var wg sync.WaitGroup
    wg.Add(1)

    quit := make(chan struct{})

    go func() {
        defer wg.Done()
        buffer := make([]byte, 0, 32768) // 32KB buffer
        for {
            select {
            case <- quit:
                return
            default:

                chunk, err := capture.ReadChunk(4096)
                if err == io.EOF {
                    fmt.Println("EOF reached, stopping capture.")
                    return
                }
                if err != nil {
                    fmt.Println("Error reading audio chunk:", err)
                    continue
                }
                
                // dont want to use for now.
                processedChunk := processor.ReduceNoise(chunk)
                if len(processedChunk) == 0 {
                    fmt.Println("Processed chunk is empty, skipping")
                    continue
                }

                buffer = append(buffer, chunk...)
                if len(buffer) >= 32768 {
                    go func(data []byte) {
                        transcript, err := transcriber.ProcessAudioChunk(data)
                        if err != nil {
                            fmt.Println("Error transcribing chunk:", err)
                        } else if transcript != "" {
                            fmt.Printf("Transcribed: %s\n", transcript)
                        }
                    }(buffer)
                }
                
                n, err := outputFile.Write(chunk)
                if err != nil {
                    fmt.Println("Error writing to file:", err)
                    return
                }
                mu.Lock()
                totalBytesWritten += n
                mu.Unlock()

                buffer = buffer[:0]
                
            }
        }
    }()

    // Wait for termination signal
    <-sigChan
    fmt.Println("\nStopping audio capture...")

    // close done chan to signal go routine finish
    close(quit)

    // stop audio capture
    stopChan := make(chan struct{})
    go func() {
        err := capture.Stop()
        if err != nil {
            fmt.Println("Error stopping audio capture:", err)
        }
        close(stopChan)
    }()

    select {
    case <-stopChan:
        fmt.Println("Audio capture stopped successfully")
    case <-time.After(10 * time.Second):
        fmt.Println("Timeout while stopping audio capture, forcing shutdown")
    }

    // wait for the audio writing goroutine to finish
    wg.Wait()
    
    // Finalize transcription
    finalTranscript := transcriber.Finalize()
    fmt.Println("final transcript:", finalTranscript)

    // Save full transcription to file
   err = os.WriteFile("transcript.txt", []byte(finalTranscript), 0644)
   if err != nil {
    fmt.Println("Error saving transcription:", err)
   }

    mu.Lock()
    err = updateWAVHeader(outputFile, totalBytesWritten)
    mu.Unlock()
    if err != nil {
        fmt.Println("Error updating WAV header:", err)
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