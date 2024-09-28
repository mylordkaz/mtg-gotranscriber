import sys
import subprocess

def transcribe(audio_path):
    result = subprocess.run(['pipx', 'run', 'openai-whisper', audio_path], capture_output=True, text=True)
    return result.stdout.strip()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python whisper_transcriber.py <audio_path>")
        sys.exit(1)
 
    audio_path = sys.argv[1]
    transcription = transcribe(audio_path)
    print(transcription)