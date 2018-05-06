package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"

	"github.com/ursiform/sleuth"
)

func main() {
	//systray.Run(onReady, onExit)
	go autodiscover()

	ip := string(GetOutboundIP())

	exec.Command("vlc", "-vvv", "dshow:// :dshow-vdev=\"None\" :dshow-adev=\"\"", "--sout",
		fmt.Sprintf("'#transcode{acodec=vorb,ab=128}:standard{access=http,mux=ogg,dst=%s:8080}'", ip))
}

/*
	Code to handle autodiscovery of service on local network
*/

type ipHandler struct{}

func autodiscover() {
	handler := new(ipHandler)

	client, err := sleuth.New(&sleuth.Config{Service: "ip-discovery", Handler: handler})
	defer client.Close()
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Ready")
	}

	http.ListenAndServe(":9872", handler)
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
