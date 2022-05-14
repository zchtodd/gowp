package main

import (
    "log"
    "flag"
    "bufio"
    "path"
    "io/ioutil"
    "net/url"
    "net/http"
    "github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "")

func startProxy(proxyPassURL *url.URL) {
   proxyServerURL := url.URL{Scheme: "ws", Host: *addr, Path: "/proxy"}

   c, _, err := websocket.DefaultDialer.Dial(proxyServerURL.String(), nil)
   if err != nil {
       log.Fatal("dial", err)
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

       u, err := url.Parse(proxyPassURL.String())
       if err != nil {
           log.Println(err)
           return
       }

       u.Path = path.Join(u.Path, request.URL.String())

       request.RequestURI = ""
       request.URL = u

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
   proxyPass := flag.String("proxy-pass", "", "The server that should receive requests from the proxy.")
   flag.Parse()

   proxyPassURL, err := url.Parse(*proxyPass)
   if err != nil {
       log.Fatal(err)
   }

   startProxy(proxyPassURL)
}
