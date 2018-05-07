package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ursiform/sleuth"
)

const ffmpegPort = "7843"
const ipURL = "sleuth://streamplay-ip/ip:9872"

/*
	Functions to handle selection of audio device
*/

func printAudioDevices() {
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	out, _ := cmd.CombinedOutput()

	regex := regexp.MustCompile("(\"[A-z].*?\")")
	dev := strings.Join(regex.FindAllString(fmt.Sprintf("%s", out), -1), "\n")

	fmt.Println("Use one of these devices with the -a flag to stream audio from this device")
	fmt.Print(dev)
}

/*
	Functions to handle autodiscovery of service on local network
*/

func autodiscover(iface string) (string, error) {
	config := &sleuth.Config{
		Interface: iface,
		LogLevel:  "debug",
	}

	client, err := sleuth.New(config)
	defer client.Close()
	if err != nil {
		return "", err
	}
	log.Println("Ready")

	// Wait for server to come online
	client.WaitFor("streamplay-ip")

	// Wait for server to finish coming online
	time.Sleep(time.Second)

	req, err := http.NewRequest("GET", ipURL, nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	ip, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

/*
	Functions to implement streaming with ffmpeg
*/

func stream(vSrc, aSrc, ip string) {
	log.Printf("Starting stream to %s...\n", ip)

	// Program args for ffmpeg
	args := []string{
		// 'dshow' us used for windows only
		"-f", "dshow",

		// Inputs
		// With video: "-i", fmt.Sprintf("video='%s':audio='%s'", vSrc, aSrc),
		"-i", fmt.Sprintf("audio=%s", aSrc),
		/*
			// Video options
			"-preset", "ultrafast", "-vcodec", "libx264", "-tune", "zerolatency",
			"-r", "24", "-async", "1",
		*/
		// Audio options
		"-acodec", "libmp3lame", "-ab", "128k", "-ar", "44100",

		// Output options
		"-maxrate", "1m", "-bufsize", "3000k", "-f", "rtmp",
		fmt.Sprintf("rtp://%s:%s", ip, ffmpegPort),
	}

	stream := exec.Command("ffmpeg", args...)

	stream.Stdout = os.Stdout
	stream.Stderr = os.Stderr
	err := stream.Start()
	if err != nil {
		log.Print(err)
	}
	stream.Wait()
}

// main starts the autodiscovery server, parses flags, and begins streaming
func main() {
	var (
		listAudio, listVideo bool
		aSrc, vSrc, iface    string
	)

	flag.BoolVar(&listAudio, "list-audio", false, "Lists available audio devices")
	flag.BoolVar(&listVideo, "list-video", false, "UNDEFINED: Lists available video devices")
	flag.StringVar(&aSrc, "a", "", "Audio device to stream")
	flag.StringVar(&vSrc, "v", "", "Video device to use")
	flag.StringVar(&iface, "iface", "Wi-Fi", "Network interface on which to listen")

	flag.Parse()

	for {
		// Start autodiscovery server
		ip, err := autodiscover(iface)
		if err != nil {
			log.Print("error in autodiscovery: ", err)
		}

		if listAudio {
			printAudioDevices()
		} else {
			go stream(vSrc, aSrc, ip)
		}
	}
}
