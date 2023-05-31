package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	NGROK_WEB_INTERFACE_HOST = "127.0.0.1"
	NGROK_WEB_INTERFACE_PORT = 4040
)

type ngrokTunnelsApiResponse struct {
	Tunnels []struct {
		PublicURL string `json:"public_url"`
		Config    struct {
			Addr string `json:"addr"`
		} `json:"config"`
	} `json:"tunnels"`
}

func makeCommandFlags() {
	flag.StringVar(
		&NGROK_WEB_INTERFACE_HOST,
		"ngrok Web Interface Host",
		NGROK_WEB_INTERFACE_HOST,
		"The host of the ngrok web interface",
	)
	flag.IntVar(
		&NGROK_WEB_INTERFACE_PORT,
		"ngrok Web Interface Port",
		NGROK_WEB_INTERFACE_PORT,
		"The port of the ngrok web interface",
	)
	flag.Parse()
}

func getNgrokTunnelUrls(url string) (ngrokUrls []string, originalUrls []string, err error) {
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("Please make sure you have atleast one instance of ngrok running")
		return
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	var response ngrokTunnelsApiResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return
	}

	for _, tunnel := range response.Tunnels {
		ngrokUrls = append(ngrokUrls, tunnel.PublicURL)
		originalUrls = append(originalUrls, tunnel.Config.Addr)
	}

	return
}

func getUrlHostsAndPorts(urls []string) (hostnames []string, ports []string, err error) {
	for _, urlString := range urls {
		var parsedUrl *url.URL
		parsedUrl, err = url.Parse(urlString)
		if err != nil {
			return
		}

		hostnames = append(hostnames, parsedUrl.Hostname())
		ports = append(ports, parsedUrl.Port())
	}

	return
}

func lookupHostnames(hostnames []string) (ips []net.IP, err error) {
	for _, hostname := range hostnames {
		var hostnameIps []net.IP
		hostnameIps, err = net.LookupIP(hostname)
		if err != nil {
			return
		}

		if len(hostnameIps) == 0 {
			err = fmt.Errorf("when resolving %s, no ips could be found", hostname)
			return
		}

		ips = append(ips, hostnameIps[0])
	}

	return
}

func getOutputInformation(originalUrls []string, ips []net.IP, ports []string) string {
	parts := []string{}
	for i := range ips {
		parts = append(parts, fmt.Sprintf("Converted URL for %s: %s:%s", originalUrls[i], ips[i], ports[i]))
	}
	return strings.Join(parts, "\n")
}

func main() {
	makeCommandFlags()

	ngrokTunnelsApiUrl := fmt.Sprintf("http://%s:%d/api/tunnels", NGROK_WEB_INTERFACE_HOST, NGROK_WEB_INTERFACE_PORT)
	ngrokUrls, originalUrls, err := getNgrokTunnelUrls(ngrokTunnelsApiUrl)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	hostnames, ports, err := getUrlHostsAndPorts(ngrokUrls)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ips, err := lookupHostnames(hostnames)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	output := getOutputInformation(originalUrls, ips, ports)
	fmt.Println(output)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
