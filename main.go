package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/kkdai/youtube/v2"
)

var (
	videoID	string
	choices	[]string
	videoFileName string
	filePath string
	wavFileName string
	selectedModel string
)

type TranscriptionResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	// Add other fields from verbose_json response as needed
}

func main() {	

huh.NewMultiSelect[string]().
    Options(
        huh.NewOption("Extract Video", "video"),
        huh.NewOption("Extract audio", "audio"),
        huh.NewOption("Transcribe audio", "transcribe"),
    ).
    Title("What actions would you like to take?").
    Value(&choices).
    Run()

for _, choice := range choices{
	switch choice{
	case "video":
		huh.NewInput().Title("Enter a Youtube ID :").Value(&videoID).Run()
		videoFileName = extractVideo(videoID)
	case "audio":
		if videoFileName == ""{
			huh.NewFilePicker().CurrentDirectory("./").Picking(true).Height(50).ShowPermissions(false).AllowedTypes([]string{"mp4"}).Value(&filePath).Title("Select a video file to convert to WAV").Run()
			wavFileName = strings.Replace(filePath, ".mp4", ".wav", 1)
			convertToWav(filePath, wavFileName)
		} else {
			wavFileName = strings.Replace(videoFileName, ".mp4", ".wav", 1)
			convertToWav(videoFileName, wavFileName)
		}
	case "transcribe":
		// Get available models
		modelOptions, err := getAvailableModels()
		if err != nil {
			log.Printf("Error getting models: %v", err)
			return
		}
		
		// Select model
		huh.NewSelect[string]().
			Title("Select a Whisper model").
			Description("Larger models are more accurate but slower").
			Options(modelOptions...).
			Value(&selectedModel).
			Run()
		
		err = transcribeWithWhisper()
		if err != nil {
			log.Printf("Transcription failed: %v", err)
		}
	}
}
}

func extractVideo(videoID string) string {
	client := youtube.Client{}
	
	video, err := client.GetVideo(videoID)
	if err != nil {
		panic(err)
	}
	
	formats := video.Formats.WithAudioChannels() // only get videos with audio
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		panic(err)
	}
	defer stream.Close()
	
	fileName := fmt.Sprintf("%s.mp4", video.Title)
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	
	_, err = io.Copy(file, stream)
	if err != nil {
		panic(err)
	}
	
	return fileName
}

func convertToWav(inputPath, outputPath string) {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputPath,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		outputPath,
	)

	err := cmd.Run()
	if err != nil {
		panic("failed to run ffmpeg")
	}
}

// getAvailableModels scans the models directory and returns available whisper models
func getAvailableModels() ([]huh.Option[string], error) {
	modelsDir := "whisper.cpp/models"
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read models directory: %v", err)
	}
	
	var modelOptions []huh.Option[string]
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "ggml-") && strings.HasSuffix(entry.Name(), ".bin") {
			// Extract model name (e.g., "ggml-tiny.en.bin" -> "tiny.en")
			modelName := strings.TrimPrefix(entry.Name(), "ggml-")
			modelName = strings.TrimSuffix(modelName, ".bin")
			
			// Get file size for display
			info, _ := entry.Info()
			sizeStr := fmt.Sprintf("%.1f MB", float64(info.Size())/(1024*1024))
			
			displayName := fmt.Sprintf("%s (%s)", modelName, sizeStr)
			fullPath := fmt.Sprintf("%s/%s", modelsDir, entry.Name())
			
			modelOptions = append(modelOptions, huh.NewOption(displayName, fullPath))
		}
	}
	
	if len(modelOptions) == 0 {
		return nil, fmt.Errorf("no whisper models found in %s", modelsDir)
	}
	
	return modelOptions, nil
}

// transcribeWithWhisper uses whisper.cpp to transcribe audio locally
func transcribeWithWhisper() error {
	// Select audio file if not already selected
	var audioPath string
	if wavFileName != "" {
		audioPath = wavFileName
	} else {
		huh.NewFilePicker().
			CurrentDirectory("./").
			Picking(true).
			Height(50).
			ShowPermissions(false).
			AllowedTypes([]string{"mp3", "wav"}).
			Value(&audioPath).
			Title("Select an audio file to transcribe").
			Run()
	}
	
	// Convert to WAV format if needed (whisper requires WAV)
	wavPath := audioPath
	if strings.HasSuffix(audioPath, ".mp3") || strings.HasSuffix(audioPath, ".mp4") {
		// Determine output extension
		ext := ".mp3"
		if strings.HasSuffix(audioPath, ".mp4") {
			ext = ".mp4"
		}
		wavPath = strings.Replace(audioPath, ext, ".wav", 1)
		log.Printf("Converting %s to WAV format...", audioPath)
		convertToWav(audioPath, wavPath)
	}
	
	log.Printf("Transcribing %s with model %s...", wavPath, selectedModel)
	
	// Call whisper binary with selected model
	cmd := exec.Command("./whisper.cpp/bindings/go/build_go/go-whisper", 
		"-model", selectedModel, 
		wavPath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("whisper transcription failed: %v\nOutput: %s", err, string(output))
	}
	
	// Filter out logs and keep only the actual transcription
	lines := strings.Split(string(output), "\n")
	var transcriptLines []string
	for _, line := range lines {
		// Keep lines that start with timestamp markers like "[0s->6.88s]"
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			transcriptLines = append(transcriptLines, line)
		}
	}
	cleanedOutput := strings.Join(transcriptLines, "\n")
	
	// Save transcription to text file
	transcriptFile := strings.Replace(wavPath, ".wav", "_transcript.txt", 1)
	if err := os.WriteFile(transcriptFile, []byte(cleanedOutput), 0644); err != nil {
		return fmt.Errorf("failed to save transcript: %v", err)
	}
	
	return nil
}
