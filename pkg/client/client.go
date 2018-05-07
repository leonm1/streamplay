package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/ursiform/sleuth"
)

const ipURL = "sleuth://streamplay-ip/ip:9872"
const ffmpegPort = "7843"

func main() {
	var iface string
	flag.StringVar(&iface, "iface", "wlan0", "Network interface on which to listen")
	flag.Parse()

	go autodiscover(iface)
	ip := GetOutboundIP()

	stream := exec.Command("ffplay", "-nodisp", fmt.Sprintf("rtp://%s:%s", ip, ffmpegPort))
	stream.Stderr = os.Stderr
	stream.Stdout = os.Stdout

	stream.Run()
}

/*
	Functions to handle autodiscovery of service on local network
*/

type ipHandler struct{}

// autodiscover locates other streamplay devices on the network and returns
// the ip of the first server found
func autodiscover(iface string) {
	handler := new(ipHandler)

	config := &sleuth.Config{
		Handler:   handler,
		Interface: iface,
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
