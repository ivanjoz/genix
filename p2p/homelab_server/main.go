package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"p2p_bridge/signal"
)

func main() {
	wsURL := os.Getenv("WS_URL")
	lambdaName := os.Getenv("LAMBDA_NAME")

	if wsURL == "" || lambdaName == "" {
		log.Fatal("WS_URL and LAMBDA_NAME environment variables are required")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	u, _ := url.Parse(wsURL)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			
			var msg signal.Msg
			json.Unmarshal(message, &msg)

			switch msg.Action {
			case "connected":
				connID := msg.Data.(string)
				log.Printf("Connected! ID: %s", connID)
				updateLambdaConfig(lambdaName, connID)
			case "offer":
				handleOffer(c, msg)
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

func updateLambdaConfig(functionName, connectionID string) {
	ctx := context.TODO()
	cfg, _ := config.LoadDefaultConfig(ctx)
	svc := lambda.NewFromConfig(cfg)
	
	res, _ := svc.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: &functionName,
	})

	variables := res.Environment.Variables
	variables["LAPTOP_ID"] = connectionID

	svc.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: &functionName,
		Environment: &lambdaTypes.Environment{
			Variables: variables,
		},
	})
	log.Printf("Lambda updated with LAPTOP_ID: %s", connectionID)
}

func handleOffer(c *websocket.Conn, msg signal.Msg) {
	pc, _ := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("Message: %s", string(msg.Data))
			d.SendText("Echo: " + string(msg.Data))
		})
	})

	offer := webrtc.SessionDescription{}
	data, _ := json.Marshal(msg.Data)
	json.Unmarshal(data, &offer)

	pc.SetRemoteDescription(offer)
	answer, _ := pc.CreateAnswer(nil)
	
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	pc.SetLocalDescription(answer)
	<-gatherComplete

	resp := signal.Msg{
		Action: "sendSignal",
		To:     msg.From,
		Data:   *pc.LocalDescription(),
	}
	payload, _ := json.Marshal(resp)
	c.WriteMessage(websocket.TextMessage, payload)
	log.Println("Sent answer to", msg.From)
}
