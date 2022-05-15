package main

import (
    "os"
    "fmt"
    "log"
    "flag"
    "strings"
    "bytes"
    "encoding/json"
    "bufio"
    "path"
    "io/ioutil"
    "net/url"
    "net/http"
    "github.com/mdp/qrterminal/v3"
    "github.com/gorilla/websocket"
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
       u, err := url.Parse(proxyPassHost + ":" + hostAndPort[1])
       if err != nil {
           log.Println(err)
           return
       }

       u.Path = path.Join(u.Path, request.URL.String())

       request.RequestURI = ""
       request.URL = u

       log.Printf("Sending proxy request: %s\n", u.String())

       client := http.Client{}
       response, err := client.Do(request)
       if err != nil {
           log.Println(err)
           return
       }

       b, err := ioutil.ReadAll(response.Body)
       if err != nil {
          log.Println(err)
          return
       }

       c.WriteMessage(websocket.TextMessage, b)
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

   qrterminal.Generate("http://" + proxyHostURL.Host, qrterminal.L, os.Stdout)
   fmt.Printf("\nYour proxy session has started!\n")
   fmt.Printf("\nScan the QR code above or go to one of the following addresses: \n\n")

   for _, proxyPassPort := range proxyPassPorts {
       fmt.Printf("    https://%s:%s\n", proxyHostURL.Hostname(), proxyPassPort)
   }

   StartProxy(proxyHostURL, *proxyPassHost)
}
