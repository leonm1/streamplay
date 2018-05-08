package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	streamPort = 7843
)

var (
	ipChan               chan string
	aSrc, vSrc, logLevel string
	streams              map[string]bool
)

/*
	Functions to handle selection of audio device
*/

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

	// Get network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println("Could not get network interfaces")
	}

	// Put interfaces into a list 'i'
	var ifaceStr string
	for i := 0; i < len(ifaces); i++ {
		ifaceStr += "\"" + ifaces[i].Name + "\"\n"
	}

	// Print out video devices followed by audio devices
	fmt.Printf("Available video devices (may support audio as well):\n%s\n\n"+
		"Available audio devices:\n%s\n\nAvailable network interfaces:\n%s\n", v, a, ifaceStr)
}

/*
	Functions to handle autodiscovery of service on local network
*/

func autodiscover(ifaceStr string) {
	/*
		Start server broadcast
	*/
	iface, err := net.InterfaceByName(ifaceStr)
	if err != nil {
		log.Fatalf("Could not find interface '%s': %s", iface.Name, err)
	}

	meta := []string{"version=0.1.0"}
	interfaces := []net.Interface{*iface}

	service, err := zeroconf.Register(
		"streamplay-server",       // service instance name
		"_streamplay-server._tcp", // service type and protocl
		"local.",                  // service domain
		streamPort,                // service port
		meta,                      // service metadata
		interfaces,                // register on ifaces listed in
	)
	defer service.Shutdown()
	if err != nil {
		panic("Error starting zeroconf browser: " + err.Error())
	}
	/*
		Get clients
	*/
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Channel to receive discovered service entries
	entries := make(chan *zeroconf.ServiceEntry)

	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Println("Found client:", entry.ServiceInstanceName(), entry.Text)
			ip := entry.AddrIPv4[0].String()

			if _, loaded := streams[ip]; !loaded {
				if vSrc == "" {
					streamAudio(aSrc, ip)
				} else {
					stream(aSrc, vSrc, ip)
				}
			}
		}
	}(entries)

	ctx := context.Background()

	for {
		err = resolver.Browse(ctx, "_streamplay-client._tcp", "local.", entries)
		if err != nil {
			log.Fatalln("Failed to browse:", err.Error())
		}
		time.Sleep(5 * time.Second)
	}

	<-ctx.Done()
}

/*
	Functions to implement streaming with ffmpeg
*/

func streamAudio(aSrc, ip string) {
	fmt.Printf("Streaming to %s:%d\n", ip, streamPort)
	streams[ip] = true

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
	stream.Stdout = os.Stdout
	stream.Stderr = os.Stderr

	err := stream.Start()
	if err != nil {
		fmt.Print(err)
	}
	stream.Wait()
	fmt.Println("Closed stream")
	delete(streams, ip)

}

func stream(aSrc, vSrc, ip string) {
	fmt.Printf("Streaming to %s:%d\n", ip, streamPort)
	streams[ip] = true

	if aSrc == "" {
		// Duplicate video source for audio
		aSrc = vSrc
	}

	// Program args for ffmpeg
	args := []string{
		// 'dshow' us used for windows only
		"-f", "dshow",

		// Inputs
		"-i", fmt.Sprintf("video='%s':audio='%s'", vSrc, aSrc),

		// Video options
		"-preset", "ultrafast", "-vcodec", "libx264", "-tune", "zerolatency",
		"-r", "24", "-async", "1",

		// Audio options
		"-acodec", "libmp3lame", "-ab", "128k", "-ar", "44100",

		// Output options
		"-maxrate", "1m", "-bufsize", "3000k", "-f", "rtp",
		fmt.Sprintf("rtp://%s:%d", ip, streamPort),
	}

	stream := exec.Command("ffmpeg", args...)
	stream.Stdout = os.Stdout
	stream.Stderr = os.Stderr

	err := stream.Start()
	if err != nil {
		fmt.Print(err)
	}
	stream.Wait()
	fmt.Println("Closed stream")
	delete(streams, ip)
}

// main starts the autodiscovery server, parses flags, and begins streaming
func main() {
	var (
		listDev  bool
		ifaceStr string
	)

	flag.BoolVar(&listDev, "dev", false, "Lists available input devices")
	flag.StringVar(&ifaceStr, "iface", "Wi-Fi", "Network interface on which to listen")
	flag.StringVar(&aSrc, "a", "", "Audio device to stream")
	flag.StringVar(&vSrc, "v", "", "Video device to use")
	flag.StringVar(&logLevel, "d", "silent", "Log level for sleuth ('debug', 'error', 'warn', or 'silent')")
	flag.Parse()

	if listDev {
		printDev()
	} else {
		ipChan = make(chan string)

		// Default to audio streaming if no video device specified
		if aSrc == "" && vSrc == "" {
			fmt.Println("You must specify an audio device or a video device to stream with the -a or -v flags")
			fmt.Println("Available devices:")
			printDev()
			flag.Usage()
		} else if vSrc == "" {
			streams = make(map[string]bool)
			autodiscover(ifaceStr)
		}
	}
}
