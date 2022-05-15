package main

import (
    "log"
    "fmt"
    "bytes"
    "strings"
    "net/http"
    "encoding/json"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/lithammer/shortuuid/v4"
)

type RegisterPayload struct {
   Ports []string `json:"ports"`
}

type ProxyValue struct {
    Conn    *websocket.Conn
    Channel chan []byte
}

type Proxy struct {
    Router  *mux.Router
    Clients map[string]*ProxyValue
    Ports   []string
}

type ProxyHandler func(*Proxy, http.ResponseWriter, *http.Request)

var upgrader = websocket.Upgrader{}

func WrapHandler(p *Proxy, handler ProxyHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        handler(p, w, r)
    }
}

func handleRoot(p *Proxy, w http.ResponseWriter, r *http.Request) {
    hostComponents := strings.Split(r.Host, ".")
    subdomain := hostComponents[0]

    log.Printf("Received request to proxy for subdomain: %s\n", subdomain)

    if proxyValue, ok := p.Clients[subdomain]; ok {
        buf := &bytes.Buffer{}
        r.Write(buf)
        
        log.Printf("Proxying request\n")
        proxyValue.Conn.WriteMessage(websocket.TextMessage, buf.Bytes())

        proxyResponse := <- proxyValue.Channel
        w.Write(proxyResponse)
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

    p.Clients[subdomain] = &ProxyValue{Conn: nil, Channel: make(chan []byte)}
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

    defer conn.Close()

    if proxyValue, ok := p.Clients[subdomain]; ok {
        proxyValue.Conn = conn
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                log.Printf("err: %v\n", err)
                break
            }
    
            log.Printf("Writing proxy response to channel\n")
            proxyValue.Channel <- message
        }
    } else {
        log.Printf("Subdomain not found: %s\n", subdomain)
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
