package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mdp/qrterminal/v3"
)

type arrayFlags []string

func (a *arrayFlags) String() string {
	return ""
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

type RegisterPayload struct {
	Ports []string `json:"ports"`
}

func Register(proxyPassPorts arrayFlags) string {
	payload := RegisterPayload{Ports: proxyPassPorts}
	payloadBuf := new(bytes.Buffer)

	json.NewEncoder(payloadBuf).Encode(payload)

	request, err := http.NewRequest("POST", "http://corra.io:8080/register", payloadBuf)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	response.Body.Close()
	return string(body)
}

func StartProxy(proxyHostURL *url.URL, proxyPassHost string) {
	c, _, err := websocket.DefaultDialer.Dial(proxyHostURL.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	c.EnableWriteCompression(true)
	defer c.Close()

	for {
		_, msgReader, err := c.NextReader()
		if err != nil {
			log.Println(err)
			return
		}

		request, err := http.ReadRequest(bufio.NewReader(msgReader))
		if err != nil {
			log.Println(err)
			return
		}

		hostAndPort := strings.Split(request.Host, ":")
		u, err := url.Parse(proxyPassHost + ":" + hostAndPort[1] + request.URL.String())
		if err != nil {
			log.Println(err)
			return
		}

		request.Header.Set("Accept-Encoding", "")

		request.RequestURI = ""
		request.URL = u

		log.Printf("Sending proxy request: %s\n", u.String())
		startTime := time.Now()

		client := http.Client{Timeout: time.Second * 2}
		response, err := client.Do(request)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Proxy response status code: %d\n", response.StatusCode)
		if response.StatusCode == http.StatusSwitchingProtocols {
			log.Println("Ignoring request to open websocket connection")
			c.WriteMessage(websocket.TextMessage, []byte{})
			continue
		}

		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Proxy request finished, time elapsed: %s\n", time.Since(startTime))

		startTime = time.Now()

		var compressBuf bytes.Buffer
		gz, err := gzip.NewWriterLevel(&compressBuf, gzip.BestCompression)
		if _, err := gz.Write(b); err != nil {
			log.Printf("Error compressing message, %v\n", err)
		}

		gz.Close()

		c.WriteMessage(websocket.TextMessage, compressBuf.Bytes())
		log.Printf("Done writing to websocket, time elapsed: %s (%s)\n", u.String(), time.Since(startTime))
	}
}

func main() {
	var proxyPassPorts arrayFlags

	proxyPassHost := flag.String("hostname", "", "The hostname that should receive requests from the proxy.")
	flag.Var(&proxyPassPorts, "port", "The ports that proxy requests should be routed to.")

	flag.Parse()

	proxyHostURL, err := url.Parse(fmt.Sprintf("ws://%s.corra.io:8080/proxy", Register(proxyPassPorts)))
	if err != nil {
		log.Fatal(err)
	}

	qrterminal.Generate("http://"+proxyHostURL.Host, qrterminal.L, os.Stdout)
	fmt.Printf("\nYour proxy session has started!\n")
	fmt.Printf("\nScan the QR code above or go to one of the following addresses: \n\n")

	for _, proxyPassPort := range proxyPassPorts {
		fmt.Printf("    https://%s:%s\n", proxyHostURL.Hostname(), proxyPassPort)
	}

	go StartProxy(proxyHostURL, *proxyPassHost)
	go StartProxy(proxyHostURL, *proxyPassHost)
	go StartProxy(proxyHostURL, *proxyPassHost)
	go StartProxy(proxyHostURL, *proxyPassHost)
	go StartProxy(proxyHostURL, *proxyPassHost)
	go StartProxy(proxyHostURL, *proxyPassHost)

	select {}
}
