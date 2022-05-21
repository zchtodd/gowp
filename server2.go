package main

import (
    "fmt"
    "log"
    "bytes"
    "math"
    "time"
    "strings"
    "net/http"
    "encoding/json"
    "github.com/gorilla/websocket"
    "github.com/gorilla/mux"
    "github.com/lithammer/shortuuid/v4"
)

type RegisterPayload struct {
    Ports []string `json:"ports"`
}

type Connection struct {
    WSConn *websocket.Conn
    Queued [](chan []byte)
}

type Client struct {
    Connections []*Connection
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

func handleRoot(p *Proxy, w http.ResponseWriter, r *http.Request) {
    hostComponents := strings.Split(r.Host, ".")
    subdomain := hostComponents[0]

    log.Printf("Received request to proxy for subdomain: %s\n", subdomain)

    if client, ok := p.Clients[subdomain]; ok {
        buf := &bytes.Buffer{}
        r.Write(buf)

        proxyChan := make(chan []byte)

        minConn, minQueued := client.Connections[0], math.MaxInt32
        for _, connection := range client.Connections {
            if len(connection.Queued) < minQueued {
                minQueued = len(connection.Queued)
                minConn = connection
            }
        }

        minConn.Queued = append(minConn.Queued, proxyChan)
        minConn.WSConn.WriteMessage(websocket.TextMessage, buf.Bytes())

        select {
            case proxyResponse := <- proxyChan:
                w.Write(proxyResponse)
            case <-time.After(time.Second * 30):
                log.Printf("Timed out waiting for proxy client to respond: %s\n", r.URL.String())
        }
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
    hostComponents := strings.Split(r.Host, ".")
    subdomain := hostComponents[0]

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("err: %v\n", err)
        return
    }

    connection := &Connection{
        WSConn: conn, 
        Queued: make([](chan []byte), 0),
    }

    client, ok := p.Clients[subdomain]
    if !ok {
        client = &Client{Connections: []*Connection{}}
    }

    client.Connections = append(client.Connections, connection) 
    p.Clients[subdomain] = client

    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Printf("err: %v\n", err)
            break
        }

        log.Printf("Writing proxy response to channel\n")

        connection.Queued[0] <- message
        connection.Queued = connection.Queued[1:]
    }
}

func main() {
    r := mux.NewRouter()

    proxy := &Proxy{Router: r, Clients: make(map[string]*Client), Ports: []string{"8080"}}

    r.HandleFunc("/register", WrapHandler(proxy, handleRegister)).Methods("POST")
    r.HandleFunc("/proxy", WrapHandler(proxy, handleProxy)).Methods("GET")
    r.PathPrefix("/").HandlerFunc(WrapHandler(proxy, handleRoot))

    http.ListenAndServe(":8080", r)
}
