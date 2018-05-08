package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/zeromq/gyre/beacon"
)

const (
	ipURL         = "sleuth://streamplay-ip/ip:9872"
	ffmpegPort    = "7843"
	discoveryPort = 9872
)

var (
	logLevel string
	jobs     chan bool
)

func main() {
	var iface string
	jobs = make(chan bool)

	flag.StringVar(&iface, "iface", "wlan0", "Network interface on which to listen")
	flag.StringVar(&logLevel, "d", "silent", "Log level for sleuth ('debug', 'error', 'warn', or 'silent')")
	flag.Parse()

	go autodiscover(iface)
	ip := GetOutboundIP()

	for range jobs {
		stream := exec.Command("ffplay", "-nodisp", "-rtsp_flags", "listen",
			fmt.Sprintf("rtsp://%s:%s", ip, ffmpegPort))

		stream.Stderr = os.Stderr
		stream.Stdout = os.Stdout

		stream.Run()
	}
}

/*
	Functions to handle autodiscovery of service on local network
*/

type ipHandler struct{}

// autodiscover locates other streamplay devices on the network and returns
// the ip of the first server found
func autodiscover(iface string) {
	b := beacon.New()
	b = b.SetInterface(iface)
	b = b.SetPort(discoveryPort)

	b.Publish([]byte("Client"))

	signals := b.Signals()

	<-signals
}

// ipHandler's ServeHTTP responds to any
func (h *ipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body.Close()
	body := GetOutboundIP()
	fmt.Fprint(w, body)

	// Send signal to begin stream
	jobs <- true
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
