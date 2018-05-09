package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	streamPort = 7843 // Port over which to stream with FFmpeg
)

var (
	doneCh = make(chan bool) // Channel to signal exit of ffplay
	debug  bool              // Flag to output debug info
)

func main() {
	var (
		ifaceStr string // String containing the name of the interface to broadcast over zeroconf
		listDev  bool   // Flag to display network devices
	)

	flag.StringVar(&ifaceStr, "iface", "W-Fi", "Network interface on which to listen")
	flag.BoolVar(&listDev, "dev", false, "Lists available network interfaces")
	flag.BoolVar(&debug, "d", false, "Debug: output debug information")
	flag.Parse()

	// If dev flag is present, quit after displaying available devices
	if listDev {
		printDev()
	} else {
		// Start zeroconf server
		go autodiscover(ifaceStr)
		ip := GetOutboundIP()

		stream := exec.Command("ffplay", "-autoexit", "-nodisp", "-rtsp_flags", "listen", fmt.Sprintf("rtsp://%s:%d", ip, streamPort))

		// If in debug mode, set ffplay output to stdout
		if debug {
			stream.Stderr = os.Stderr
			stream.Stdout = os.Stdout
		}

		// Restart ffplay and wait for new stream in case stream fails
		for {
			fmt.Println("Waiting for stream...")
			stream.Run()
			fmt.Println("Finished stream")

			// Wait before starting opening ffplay again
			time.Sleep(time.Second)
		}
	}
}

/*
	Functions to handle autodiscovery of service on local network
*/

func autodiscover(ifaceStr string) {
	/*
		Start client broadcast
	*/
	iface, err := net.InterfaceByName(ifaceStr)
	if err != nil {
		log.Fatalf("Could not find interface '%s': %s", iface.Name, err)
	}

	meta := []string{"version=0.1.0"}
	interfaces := []net.Interface{*iface}

	service, err := zeroconf.Register(
		"streamplay-client",       // service instance name
		"_streamplay-client._tcp", // service type and protocl
		"local.",                  // service domain
		streamPort,                // service port
		meta,                      // service metadata
		interfaces,                // register on ifaces listed in
	)
	defer service.Shutdown()
	if err != nil {
		panic("Error starting zeroconf browser: " + err.Error())
	}

	<-doneCh
	fmt.Println("Stopping zeroconf server")
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

// printDev prints available network interfaces
func printDev() {
	// Get network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Could not get network interfaces")
	}

	// Parse slice into quotation-mark- and newline-delimited string
	var ifaceStr string
	for i := 0; i < len(ifaces); i++ {
		ifaceStr += "\"" + ifaces[i].Name + "\"\n"
	}

	fmt.Printf("Available network interfaces:\n%s\n", ifaceStr)
}
