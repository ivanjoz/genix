package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"p2p_bridge/config"
	"p2p_bridge/homelab_server/install"
	sigmsg "p2p_bridge/signal"
)

var (
	installFlag   = flag.Bool("install", false, "Install the server as a systemd service and start it")
	uninstallFlag = flag.Bool("uninstall", false, "Uninstall the systemd service and remove the binary")
)

func main() {
	flag.Parse()

	// Handle install command
	if *installFlag {
		if err := install.RunInstall(); err != nil {
			log.Fatalf("Installation failed: %v", err)
		}
		log.Println("Installation completed successfully!")
		log.Printf("Service status can be checked with: systemctl status %s", install.ServiceName)
		return
	}

	// Handle uninstall command
	if *uninstallFlag {
		if err := install.RunUninstall(); err != nil {
			log.Fatalf("Uninstallation failed: %v", err)
		}
		log.Println("Uninstallation completed successfully!")
		return
	}

	// Normal operation
	cfg := config.GetDefaultConfig()
	wsURL := cfg.WebSocketURL

	if wsURL == "" {
		log.Fatal("WEBSOCKET_URL is required in credentials.json")
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

			var msg sigmsg.Msg
			json.Unmarshal(message, &msg)

			switch msg.Action {
			case "connected":
				connID := msg.Data.(string)
				log.Printf("Connected! ID: %s", connID)
				updateLambdaConfig(cfg, connID)
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

func updateLambdaConfig(cfg *config.Config, connectionID string) {
	ctx := context.TODO()
	awsCfg, err := awssdkconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return
	}

	svc := lambda.NewFromConfig(awsCfg)

	lambdaName := cfg.GetLambdaFunctionName()
	res, err := svc.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: &lambdaName,
	})
	if err != nil {
		log.Printf("Failed to get Lambda config: %v", err)
		return
	}

	variables := res.Environment.Variables
	variables["LAPTOP_ID"] = connectionID

	_, err = svc.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: &lambdaName,
		Environment: &lambdaTypes.Environment{
			Variables: variables,
		},
	})
	if err != nil {
		log.Printf("Failed to update Lambda config: %v", err)
		return
	}

	log.Printf("Lambda updated with LAPTOP_ID: %s", connectionID)
}

func handleOffer(c *websocket.Conn, msg sigmsg.Msg) {
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

	resp := sigmsg.Msg{
		Action: "sendSignal",
		To:     msg.From,
		Data:   *pc.LocalDescription(),
	}
	payload, _ := json.Marshal(resp)
	c.WriteMessage(websocket.TextMessage, payload)
	log.Println("Sent answer to", msg.From)
}
