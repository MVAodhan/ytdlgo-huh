package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	_ "github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
)

var (
	videoID	string
	groqAPI string
	choices	[]string
	videoFileName string
	filePath string
	wavFileName string
)

type TranscriptionResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	// Add other fields from verbose_json response as needed
}

func main() {	

	



// 	err := godotenv.Load()
//   if err != nil {
//     log.Fatal("Error loading .env file")
//   }
//   groqAPI = os.Getenv("GROQ_API")

//   if groqAPI == "" {
// 		log.Fatal("GROQ_API_KEY environment variable is not set")
// 	}
// 	transcribeAuidio(fmt.Sprintf("./%s",wavFileName))

huh.NewMultiSelect[string]().
    Options(
        huh.NewOption("Extract Video", "video"),
        huh.NewOption("Extract audio", "audio"),
    ).
    Title("What actions would you like to take ").
    Value(&choices).
    Run()

for _, choice := range choices{
	switch choice{
	case "video":
		huh.NewInput().Title("Enter a Youtube ID :").Value(&videoID).Run()
		videoFileName = extractVideo(videoID)
	case "audio":
		if videoFileName == ""{
			huh.NewFilePicker().CurrentDirectory("./").Picking(true).Height(50).ShowPermissions(false).AllowedTypes([]string{"mp4"}).Value(&filePath).Title("Select a video file to convert to an mp3").Run()
			wavFileName = strings.Replace(filePath, "mp4", "mp3", 1)
			convertToWav(filePath, wavFileName)
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

func convertToWav(inputPath, outDir string) {
	cmd := exec.Command("ffmpeg",
	"-y",
	"-i", inputPath,
	"-ac", "1",
	"-ar", "16000",
		outDir,
	)

	err := cmd.Run()
	if err != nil {
		panic("failed to run ffmpeg")
	}
}

func transcribeAuidio(audioFileLocation string) error {
    // Create a buffer to write our multipart form data
   

   file, err := os.Open(audioFileLocation)
	if err != nil {
		log.Fatalf("Failed to open audio file: %v", err)
	}
	defer file.Close()

	// Create a buffer to write our multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the file field
	filePart, err := writer.CreateFormFile("file", "audio.m4a")
	if err != nil {
		log.Fatalf("Failed to create form file: %v", err)
	}
	_, err = io.Copy(filePart, file)
	if err != nil {
		log.Fatalf("Failed to copy file content: %v", err)
	}

	// Add other form fields
	writer.WriteField("model", "whisper-large-v3-turbo")
	writer.WriteField("temperature", "0")
	writer.WriteField("response_format", "verbose_json")

	// Close the writer to finalize the multipart message
	err = writer.Close()
	if err != nil {
		log.Fatalf("Failed to close writer: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/audio/transcriptions", &requestBody)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+groqAPI)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var transcription TranscriptionResponse
	err = json.Unmarshal(body, &transcription)
	if err != nil {
		log.Fatalf("Failed to parse response: %v", err)
	}

	// Print the transcription text
	fmt.Println(transcription.Text)

    return nil
}
