package main

import (
    "fmt"
    "log"
    "net/http"
    "p2p-chat-app/internal/chat"
    "p2p-chat-app/internal/network"
)

func main() {
    fmt.Println("Starting P2P Chat Application...")

    c := chat.Chat{}
    go c.StartChat()

    n := network.NewPeer()
    go n.Connect()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Welcome to the P2P Chat Application!")
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}