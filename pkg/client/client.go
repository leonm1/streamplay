package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	"github.com/ursiform/sleuth"
)

const ipURL = "sleuth://streamplay-ip/ip:9872"
const ffmpegPort = "7843"

func main() {
	var iface string
	flag.StringVar(&iface, "iface", "eth0", "Network interface on which to listen")
	flag.Parse()

	ip, err := autodiscover(iface)
	if err != nil {
		log.Panic(err)
	}

	log.Print("Found IP: ", ip)

	vlc := exec.Command("ffplay", "-nodisp", fmt.Sprintf("rtp://%s:%s", ip, ffmpegPort))

	out, err := vlc.CombinedOutput()
	fmt.Printf("%s", out)
	if err != nil {
		log.Panic(err)
	}
}

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
