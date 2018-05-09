package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	streamPort = 7843 // Port over which to stream with FFmpeg
)

var (
	aSrc, vSrc string          // Sources for stream set via cli flags
	streams    map[string]bool // Index of in-progress streams
	debug      bool            // Flag to output debug info
)

/*
	Functions to handle selection of audio device
*/

// printDev prints the audio/video devices available through DirectShow
// accessible to FFmpeg
func printDev() {
	// Regexp to extract device names
	regex := regexp.MustCompile("(\"[A-z].*?\")")

	// Pull input devices from ffmpeg
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	out, _ := cmd.CombinedOutput()

	// Split audio and video devices: video will be outSl[0] and audio will be outSl[1]
	outSl := strings.Split(string(out), "DirectShow audio devices")

	// Insert newline after each device
	v := strings.Join(regex.FindAllString(fmt.Sprintf("%s", outSl[0]), -1), "\n")
	a := strings.Join(regex.FindAllString(fmt.Sprintf("%s", outSl[1]), -1), "\n")

	// Print out video devices followed by audio devices
	fmt.Printf("Available video devices (may support audio as well):\n%s\n\n"+
		"Available audio devices:\n%s\n", v, a)
}

/*
	Functions to handle autodiscovery of service on local network
*/

// autodiscover browses the LAN for zeroconf clients to stream,
// ensures there is no existing stream to the client, and opens a new stream
func autodiscover() {
	/*
		Get clients
	*/
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Channel to receive discovered service entries
	entries := make(chan *zeroconf.ServiceEntry)

	// Service discovery handler
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			ip := entry.AddrIPv4[0].String()

			// If ip is logged in map, another stream is already in progress to client
			if _, loaded := streams[ip]; !loaded {
				fmt.Println("Found new client:", ip)

				// Default to audio-only streaming if video source is empty
				if vSrc == "" {
					go streamAudio(aSrc, ip)
				} else {
					go stream(aSrc, vSrc, ip)
				}
			}
		}
	}(entries)

	ctx := context.Background()

	// Search for new clients every five seconds
	for {
		// Search for available clients
		err = resolver.Browse(ctx, "_streamplay-client._tcp", "local.", entries)
		if err != nil {
			log.Fatalln("Failed to browse:", err.Error())
		}

		// Wait five seconds in between each search for clients
		time.Sleep(5 * time.Second)
	}

	<-ctx.Done()
}

/*
	Functions to implement streaming with ffmpeg
*/

// streamAudio opens an audio-only stream from 'aSrc' to the
// 'ip' parameter on const 'streamPort'
func streamAudio(aSrc, ip string) {
	fmt.Printf("Opening stream to %s:%d\n", ip, streamPort)

	// Log in progress streams to prevent multiple streams to the same source
	streams[ip] = true
	defer delete(streams, ip)

	// Program args for ffmpeg
	args := []string{
		// 'dshow' us used for windows only
		"-f", "dshow",

		// Inputs
		"-i", fmt.Sprintf("audio=%s", aSrc),

		// Audio options
		"-acodec", "libmp3lame", "-ab", "128k", "-ar", "44100",

		// Output options
		"-maxrate", "1m", "-bufsize", "3000k", "-f", "rtsp", "-rtsp_transport", "tcp",
		fmt.Sprintf("rtsp://%s:%d", ip, streamPort),
	}

	stream := exec.Command("ffmpeg", args...)
	if debug {
		stream.Stdout = os.Stdout
		stream.Stderr = os.Stderr
	}

	err := stream.Run()
	if err != nil {
		fmt.Print(err)
	}

	fmt.Println("Closed stream to", ip)
}

// stream opens an video stream from 'vSrc' to the
// 'ip' parameter on const 'streamPort'
func stream(aSrc, vSrc, ip string) {
	fmt.Printf("Opening stream to %s:%d\n", ip, streamPort)

	// Log in progress streams to prevent multiple streams to the same source
	streams[ip] = true
	defer delete(streams, ip)

	if aSrc == "" {
		// Duplicate video source for audio if audio is not present
		aSrc = vSrc
	}

	// Program args for ffmpeg
	args := []string{
		// 'dshow' is used for windows only
		"-f", "dshow",

		// Inputs
		"-i", fmt.Sprintf("video='%s':audio='%s'", vSrc, aSrc),

		// Video options
		"-preset", "ultrafast", "-vcodec", "libx264", "-tune", "zerolatency",
		"-r", "24", "-async", "1",

		// Audio options
		"-acodec", "libmp3lame", "-ab", "128k", "-ar", "44100",

		// Output options
		"-maxrate", "1m", "-bufsize", "3000k", "-f", "rtsp", "-rtsp_transport", "tcp",
		fmt.Sprintf("rtsp://%s:%d/live.sdp", ip, streamPort),
	}

	stream := exec.Command("ffmpeg", args...)
	if debug {
		stream.Stdout = os.Stdout
		stream.Stderr = os.Stderr
	}

	err := stream.Run()
	if err != nil {
		fmt.Print(err)
	}

	fmt.Println("Closed stream to", ip)
}

// main starts parses the cli flags and starts discovering and handling new clients
func main() {
	var (
		listDev bool // Flag to display sources
	)

	flag.BoolVar(&listDev, "dev", false, "Lists available input devices")
	flag.StringVar(&aSrc, "a", "", "Audio device to stream")
	flag.StringVar(&vSrc, "v", "", "Video device to use")
	flag.BoolVar(&debug, "d", false, "Output debug information")
	flag.Parse()

	if listDev {
		printDev()
	} else {
		// Output help information if no devices specified
		if aSrc == "" && vSrc == "" {
			fmt.Println("You must specify an audio device or a video device to stream with the -a or -v flags")
			fmt.Println("Available devices:")
			printDev()
			flag.Usage()
		} else {
			// Initialize global map of in-progress streams
			streams = make(map[string]bool)

			// Start autodiscovery server/stream handling
			autodiscover()
		}
	}
}
