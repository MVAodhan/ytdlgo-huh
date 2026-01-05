package main

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/kkdai/youtube/v2"
)

var (
	videoID	string
)

func main() {	
	huh.NewInput().Title("Enter a Youtube ID:").Value(&videoID).Run()
	_ = extractVideo(videoID)
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

