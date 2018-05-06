package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	"github.com/ursiform/sleuth"
)

const ipURL = "sleuth://ip-discovery/ip:9872"

func main() {
	ip, err := autodiscover()
	if err != nil {
		log.Panic(err)
	}

	log.Print("Found IP:", ip)

	vlc := exec.Command("vlc", "-vvv", fmt.Sprintf("http://%s:8080", ip))

	err = vlc.Run()
	if err != nil {
		log.Panic(err)
	}
}

func autodiscover() (string, error) {
	config := &sleuth.Config{
		Interface: "Wi-Fi",
		LogLevel:  "debug",
	}

	client, err := sleuth.New(config)
	defer client.Close()
	if err != nil {
		return "", err
	}
	log.Println("Ready")

	// Wait for server to come online
	client.WaitFor("ip-discovery")

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
