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
	streamPort = 7843
)

var doneCh = make(chan bool)

func main() {
	var (
		ifaceStr string
		listDev  bool
	)

	flag.StringVar(&ifaceStr, "iface", "wlan0", "Network interface on which to listen")
	flag.BoolVar(&listDev, "dev", false, "Lists available network interfaces")
	flag.Parse()

	if listDev {
		printDev()
	} else {
		for {
			go autodiscover(ifaceStr)

			ip := GetOutboundIP()
			stream := exec.Command("ffplay", "-autoexit", "-nodisp", "-rtsp_flags", "listen", fmt.Sprintf("rtsp://%s:%d", ip, streamPort))

			stream.Stderr = os.Stderr
			stream.Stdout = os.Stdout

			stream.Start()
			stream.Wait()
			fmt.Println("Finished stream")
			doneCh <- true
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

func printDev() {
	// Get network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Could not get network interfaces")
	}

	// Put interfaces into a list 'i'
	var ifaceStr string
	for i := 0; i < len(ifaces); i++ {
		ifaceStr += "\"" + ifaces[i].Name + "\"\n"
	}

	// Print out video devices followed by audio devices
	fmt.Printf("Available network interfaces:\n%s\n", ifaceStr)
}
