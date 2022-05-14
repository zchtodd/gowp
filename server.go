package main

import (
    "log"
    "fmt"
    "bytes"
    "net/http"
    "github.com/gorilla/websocket"
    "github.com/lithammer/shortuuid/v4"
)

type ProxyKey struct {
    APIKey    string
    Subdomain string
}

type ProxyValue struct {
    Conn    *websocket.Conn
    Channel chan []byte
}

type Proxy struct {
    Clients map[ProxyKey]ProxyValue
}

type ProxyHandler func(*Proxy, http.ResponseWriter, *http.Request)

func WrapHandler(path string, p *Proxy, handler ProxyHandler) {
    http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
        handler(p, w, r)
    })
}

func handleRoot(p *Proxy, w http.ResponseWriter, r *http.Request) {

}

func handleRegister(p *Proxy, w http.ResponseWriter, r *http.Request) {

}

func handleProxy(p *Proxy, w http.ResponseWriter, r *http.Request) {

}

var upgrader = websocket.Upgrader{}

func main() {
    var conn *websocket.Conn

    proxyResponses := make(chan []byte)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        buf := &bytes.Buffer{}
        r.Write(buf)
        
        conn.WriteMessage(websocket.TextMessage, buf.Bytes())

        proxyResponse := <- proxyResponses
        w.Write(proxyResponse)
    })

    http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("x-corra-api-key")
        subdomain := shortuuid.New()

        proxyKey := ProxyKey{APIKey: apiKey, Subdomain: subdomain}

        fmt.Fprintf(w, subdomain)
    })

    http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
        var err error
        log.Printf("host: %s\n", r.Host)

        conn, err = upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Printf("err: %v\n", err)
            return
        }

        defer conn.Close()

        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                log.Printf("err: %v\n", err)
                break
            }

            proxyResponses <- message
        }
    })

    http.ListenAndServe(":8080", nil)
}
