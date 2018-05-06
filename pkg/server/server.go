package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/ursiform/sleuth"
)

func main() {
	//systray.Run(onReady, onExit)
	autodiscover()
	/*
		ip := string(GetOutboundIP())

		vlc := exec.Command("C:\\Program Files\\VideoLAN\\VLC\\vlc.exe", "-vvv", "input_stream", "--sout",
			fmt.Sprintf("'#transcode{acodec=vorb,ab=128}:standard{access=http,mux=ogg,dst=%s:8080}'", ip))

		err := vlc.Run()
		if err != nil {
			log.Fatal(err)
		}*/
}

/*
	Code to handle autodiscovery of service on local network
*/

type ipHandler struct{}

func autodiscover() {
	handler := new(ipHandler)

	config := &sleuth.Config{
		Handler:   handler,
		Interface: "Wi-Fi",
		LogLevel:  "debug",
		Service:   "ip-discovery",
	}

	client, err := sleuth.New(config)
	defer client.Close()
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Ready")
	}

	err = http.ListenAndServe(":80", handler)
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
