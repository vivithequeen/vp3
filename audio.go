package main

import (
	"bytes"
	"image"
	"log"
	"os"

	"time"

	"github.com/dhowden/tag"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/qeesung/image2ascii/convert"
)

func getMusicLength(fp string) time.Duration {
	f, err := os.Open(fp)
	if err != nil {
		log.Println(err)
		return 0
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Println(err)
		return 0
	}

	defer streamer.Close()
	return time.Duration(streamer.Len()) * time.Second / time.Duration(format.SampleRate)
}

func swapMusicTo(fp string) {
	f, err := os.Open(fp)
	if err != nil {
		log.Println(err)
		return
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Println(err)
		return
	}

	speaker.Clear()
	speaker.Lock()
	if currentStreamer != nil {
		currentStreamer.Close()
	}
	currentStreamer = streamer
	currentSampleRate = format.SampleRate
	speaker.Unlock()

	resampled := beep.Resample(4, format.SampleRate, beep.SampleRate(44100), streamer)
	volumeCtrl := &effects.Volume{
		Streamer: resampled,
		Base:     2,
		Volume:   currentVolume,
	}
	currentVolumeCtrl = volumeCtrl
	ctrl := &beep.Ctrl{
		Streamer: volumeCtrl,
	}
	currentCtrl = ctrl
	speaker.Play(ctrl)
}

func seekTo(s time.Duration) {
	if currentStreamer == nil || currentSampleRate == 0 {
		return
	}
	speaker.Lock()
	currentStreamer.Seek(currentSampleRate.N(s))
	speaker.Unlock()
}

func setVolume(newVolume float32) {
	if currentStreamer == nil || currentVolumeCtrl == nil {
		return
	}

	speaker.Lock()
	if currentVolumeCtrl != nil {
		if newVolume <= -7.5 {
			currentVolumeCtrl.Volume = float64(-99)
		} else {
			currentVolumeCtrl.Volume = float64(newVolume)
		}

		currentVolume = float64(newVolume)
	}
	speaker.Unlock()
}

func changeCurrentSong(fp string) (albumArt, musicTitle string, songLength time.Duration) {
	f, err := os.Open(fp)
	if err != nil {
		return
	}
	defer f.Close()

	tags, err := tag.ReadFrom(f)
	if err != nil {
		return
	}

	pic := tags.Picture()
	if pic == nil {
		return
	}

	img, _, err := image.Decode(bytes.NewReader(pic.Data))
	if err != nil {
		return
	}
	size := 15
	convertOptions := convert.DefaultOptions
	convertOptions.FixedWidth = int(float64(size) * 2.5)
	convertOptions.FixedHeight = size

	converter := convert.NewImageConverter()
	s := converter.Image2ASCIIString(img, &convertOptions)
	albumArt = s
	//albumArt = coverTheme.Render(s)
	musicTitle = "\n" + tags.Title() + " - " + tags.Artist()
	songLength = getMusicLength(fp)
	go swapMusicTo(fp)
	return
}
