package main

import (
    "log"
    "fmt"
    "bytes"
    "strings"
    "net/http"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/lithammer/shortuuid/v4"
)

type ProxyValue struct {
    Conn    *websocket.Conn
    Channel chan []byte
}

type Proxy struct {
    Clients map[string]*ProxyValue
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

    if proxyValue, ok := p.Clients[subdomain]; ok {
        buf := &bytes.Buffer{}
        r.Write(buf)
        
        proxyValue.Conn.WriteMessage(websocket.TextMessage, buf.Bytes())

        proxyResponse := <- proxyValue.Channel
        w.Write(proxyResponse)
    }
}

func handleRegister(p *Proxy, w http.ResponseWriter, r *http.Request) {
    subdomain := shortuuid.New()

    p.Clients[subdomain] = &ProxyValue{Conn: nil, Channel: make(chan []byte)}
    fmt.Fprintf(w, subdomain)
}

func handleProxy(p *Proxy, w http.ResponseWriter, r *http.Request) {
    var err error

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
    
            proxyValue.Channel <- message
        }
    }
}

func main() {
    proxy := &Proxy{Clients: make(map[string]*ProxyValue)}

    r := mux.NewRouter()

    r.HandleFunc("/register", WrapHandler(proxy, handleRegister)).Methods("GET")
    r.HandleFunc("/proxy", WrapHandler(proxy, handleProxy)).Methods("GET")

    r.PathPrefix("/").HandlerFunc(WrapHandler(proxy, handleRoot))

    http.ListenAndServe(":8080", r)
}
