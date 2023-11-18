package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
    "fmt"

	"github.com/gorilla/websocket"
)

func main() {
    interrupt := make(chan os.Signal, 1) 
    signal.Notify(interrupt, os.Interrupt) 
    u := url.URL {Scheme:"wss",Host:"rpc1.unification.io",Path:"/websocket"} 
    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)  
    if err != nil{
        log.Fatal("dail: ", err) 
        return
    } 
    defer c.Close()    

    subscribe := []byte(`{ "jsonrpc": "2.0", "method": "subscribe", "id": 0, "params": { "query": "tm.event='Tx'" } }`)
    err = c.WriteMessage(websocket.TextMessage, subscribe)
    if err != nil{
        log.Fatal("Couldn't Subscribe: " , err) 
        return
    } 

    done := make(chan struct{})  

    go func(){
        defer close(done)      
        for {
            _,m,err := c.ReadMessage()
            if err != nil{
                log.Fatal("read: ", err) 
                return
            } 
            fmt.Print(string(m))
        }  
    }()
    select {
        case <- done:
        case <- interrupt:
    }
    return
}
