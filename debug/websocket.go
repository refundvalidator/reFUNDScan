package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
    "fmt"
    "encoding/json"

	"github.com/gorilla/websocket"
)

type JsonResponse struct {
    Result struct {
        Events struct {
            MessageAction []string `json:"message.action"` 
        } `json:"events"`
    } `json:"result"`
}

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

    done := make(chan string)  

    go func(){
        defer close(done)      
        log.Println("Listening for msgs...")
        for {
            _,m,err := c.ReadMessage()
            if err != nil{
                log.Fatal("read: ", err) 
                return
            } 
            var res JsonResponse
            if err := json.Unmarshal(m,&res); err != nil {
                fmt.Printf("Cannot Unmarshal")
            }
            var action string
            if len(res.Result.Events.MessageAction) >= 1 {
                action = res.Result.Events.MessageAction[0]
                if action != "/mainchain.beacon.v1.MsgRecordBeaconTimestamp" &&
                    action != "/mainchain.wrkchain.v1.MsgRecordWrkChainBlock" &&
                    action != "/cosmos.bank.v1beta1.MsgSend" {
                    fmt.Printf(string(m))
                }
            }
        }  
    }()
    select {
    case <- done:
        fmt.Printf("Done")
        return
    case <- interrupt:
        fmt.Printf("Interrupt Detected, Done")
        return
    }
}
