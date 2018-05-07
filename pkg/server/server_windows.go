package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"

	"github.com/ursiform/sleuth"
)

const ffmpegPort = "7843"

/*
	Functions to handle selection of audio device
*/

func printAudioDevices() {
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Print(err)
	}
	fmt.Printf("%s", output)
}

/*
	Functions to handle autodiscovery of service on local network
*/

type ipHandler struct{}

// autodiscover locates other streamplay devices on the network and returns
// the ip of the first server found
func autodiscover() {
	handler := new(ipHandler)

	config := &sleuth.Config{
		Handler:   handler,
		Interface: "Wi-Fi",
		LogLevel:  "debug",
		Service:   "streamplay-ip",
	}

	client, err := sleuth.New(config)
	defer client.Close()
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Ready")
	}

	err = http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatal(err)
	}
}

// ipHandler's ServeHTTP responds to any
func (h *ipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body.Close()

	body := GetOutboundIP()

	fmt.Fprint(w, body)
}

// GetOutboundIP gets the preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

/*
	Functions to implement streaming with ffmpeg
*/

func stream(vSrc, aSrc string) error {
	ip := fmt.Sprintf("%v", GetOutboundIP())

	// Program args for ffmpeg
	args := []string{
		// 'dshow' us used for windows only
		"-f", "dshow",

		// Inputs
		// With video: "-i", fmt.Sprintf("video='%s':audio='%s'", vSrc, aSrc),
		"-i", fmt.Sprintf("audio=%s", aSrc),

		// Video options
		"-preset", "ultrafast", "-vcodec", "libx264", "-tune", "zerolatency",
		"-r", "24", "-async", "1",

		// Audio options
		"-acodec", "libmp3lame", "-ab", "128k", "-ar", "44100",

		// Output options
		"-maxrate", "1m", "-bufsize", "3000k", "-f", "rtp",
		fmt.Sprintf("rtp://%s:%s", ip, ffmpegPort),
	}

	stream := exec.Command("ffmpeg", args...)

	out, err := stream.CombinedOutput()
	fmt.Printf("%s", out)
	if err != nil {
		return err
	}

	stream.Wait()

	return nil
}

// main starts the autodiscovery server, parses flags, and begins streaming
func main() {
	var (
		listAudio, listVideo bool
		aSrc, vSrc           string
		err                  error
	)

	flag.BoolVar(&listAudio, "list-audio", false, "Lists available audio devices")
	flag.BoolVar(&listVideo, "list-video", false, "UNDEFINED: Lists available video devices")
	flag.StringVar(&aSrc, "a", "none", "Audio device to stream")
	flag.StringVar(&vSrc, "v", "none", "Video device to use")

	flag.Parse()

	// Start autodiscovery server
	go autodiscover()

	if listAudio {
		printAudioDevices()
	} else {
		err = stream(vSrc, aSrc)
		if err != nil {
			log.Fatal(err)
		}
	}
}
