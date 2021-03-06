package main

import (
    "io"
    "log"
    "fmt"
    "bytes"
    "time"
    "strings"
    "net/http"
    "encoding/json"
    "compress/gzip"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/lithammer/shortuuid/v4"
)

type RegisterPayload struct {
    Ports []string `json:"ports"`
}

type ProxyConnection struct {
    Connection   *websocket.Conn
    ResponseChan chan []byte
}

type Client struct {
    Connections []chan []byte
}

type Proxy struct {
    Queued    int
    Router    *mux.Router
    Clients   map[string]*Client
    Ports     []string
}

type ProxyHandler func(*Proxy, http.ResponseWriter, *http.Request)

var upgrader = websocket.Upgrader{}

func WrapHandler(p *Proxy, handler ProxyHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        handler(p, w, r)
    }
}

func (p *Proxy) WriteMessage(subdomain string, proxyResponse chan []byte, data []byte) {
    //proxyValue.Conn.WriteMessage(websocket.TextMessage, buf.Bytes())
    if connections, ok := p.Clients[subdomain]; ok {
        for _, connChan := range 
    }
}

/* handleRoot accepts a request for a resource and proxies that request to
** the proxy running on the remote client. */
func handleRoot(p *Proxy, w http.ResponseWriter, r *http.Request) {
    hostComponents := strings.Split(r.Host, ".")
    subdomain := hostComponents[0]

    log.Printf("Received request to proxy for subdomain: %s\n", subdomain)

    if proxyValue, ok := p.Clients[subdomain]; ok {
        buf := &bytes.Buffer{}
        r.Write(buf)
        
        p.Queued += 1

        startTime := time.Now()
        log.Printf("Proxying request: %s (%d queued)\n", r.URL.String(), p.Queued)

        proxyChan := make(chan []byte)
        go p.WriteMessage(proxyChan, buf.Bytes())

        proxyResponse := <- proxyChan

        /*
        proxyValue.Conn.WriteMessage(websocket.TextMessage, buf.Bytes())

        log.Printf("Writing request to websocket took: %s\n", time.Since(startTime))
        startTime = time.Now()

        select {
            case proxyResponse := <- proxyValue.Channel:
                w.Write(proxyResponse)
                log.Printf("Response from proxy client took: %s\n", time.Since(startTime))
            case <-time.After(time.Second * 30):
                log.Printf("Timed out waiting for proxy client to respond: %s\n", r.URL.String())
        }

        p.Queued -= 1
        */
    }
}

func handleRegister(p *Proxy, w http.ResponseWriter, r *http.Request) {
    var payload RegisterPayload

    err := json.NewDecoder(r.Body).Decode(&payload)
    if err != nil {
        log.Printf("Bad client registration\n")
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    subdomain := strings.ToLower(shortuuid.New())
    log.Printf("Proxy subscriber registered at subdomain: %s\n", subdomain)

    for _, requestedPort := range payload.Ports {
        foundPort := false
        for _, activePort := range p.Ports {
            if requestedPort == activePort {
                foundPort = true
                break
            }
        }

        if !foundPort {
            log.Printf("Opening additional port for proxying: %s\n", requestedPort)

            go http.ListenAndServe(fmt.Sprintf(":%s", requestedPort), p.Router)
            p.Ports = append(p.Ports, requestedPort)
        }
    }

    fmt.Fprintf(w, subdomain)
}

func handleProxy(p *Proxy, w http.ResponseWriter, r *http.Request) {
    var err error

    log.Printf("Opening proxy websocket connection\n")

    hostComponents := strings.Split(r.Host, ".")
    subdomain := hostComponents[0]

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("err: %v\n", err)
        return
    }

    var Client *client
    if client, ok := p.Clients[subdomain]; !ok {
        responseChan := make(chan []byte, 8)
        
        client = &Client{Connection: conn, }
    } else {
        client = p.Clients[subdomain]
    }

    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Printf("err: %v\n", err)
            break
        }

        log.Printf("Writing proxy response to channel\n")
        proxyValue.Channel <- message
    }
}

func main() {
    r := mux.NewRouter()

    proxy := &Proxy{Router: r, Clients: make(map[string]*ProxyValue), Ports: []string{"8080"}}

    r.HandleFunc("/register", WrapHandler(proxy, handleRegister)).Methods("POST")
    r.HandleFunc("/proxy", WrapHandler(proxy, handleProxy)).Methods("GET")
    r.PathPrefix("/").HandlerFunc(WrapHandler(proxy, handleRoot))

    http.ListenAndServe(":8080", r)
}
