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
	wsURL := cfg.SignalingEndpoint
	log.Printf("DEBUG: Loaded config. Signaling endpoint: %s", wsURL)

	if wsURL == "" {
		log.Fatal("ERROR: SIGNALING_ENDPOINT is required in credentials.json")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	log.Printf("DEBUG: Attempting to connect to signaling server at %s...", wsURL)
	u, err := url.Parse(wsURL)
	if err != nil {
		log.Fatalf("ERROR: Failed to parse signaling URL: %v", err)
	}

	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil {
			log.Fatalf("ERROR: Dial failed with status %d: %v", resp.StatusCode, err)
		}
		log.Fatalf("ERROR: Dial failed: %v", err)
	}
	log.Println("DEBUG: Successfully connected to signaling server.")
	defer c.Close()

	// Request identity from Lambda
	identifyMsg := sigmsg.Msg{
		Action: "sendSignal",
		To:     "me",
		Data:   "identify",
	}
	log.Println("DEBUG: Sending 'identify' request to signaling server...")
	identifyPayload, _ := json.Marshal(identifyMsg)
	if err := c.WriteMessage(websocket.TextMessage, identifyPayload); err != nil {
		log.Printf("ERROR: Failed to send identify message: %v", err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("DEBUG: WebSocket read error:", err)
				return
			}

			log.Printf("DEBUG: Received message: %s", string(message))
			var msg sigmsg.Msg
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("ERROR: Failed to unmarshal message: %v", err)
				continue
			}

			switch msg.Action {
			case "connected":
				connID, ok := msg.Data.(string)
				if !ok {
					log.Printf("ERROR: 'connected' action missing connection ID string. Got: %v", msg.Data)
					continue
				}
				log.Printf("DEBUG: Identity confirmed. Connection ID: %s", connID)
				updateLambdaConfig(cfg, connID)
			case "offer":
				log.Printf("DEBUG: Received WebRTC offer from %s", msg.From)
				handleOffer(c, msg)
			default:
				log.Printf("DEBUG: Received unhandled action: %s", msg.Action)
			}
		}
	}()

	for {
		select {
		case <-done:
			log.Println("DEBUG: Background reader exited.")
			return
		case <-interrupt:
			log.Println("DEBUG: Interrupt signal received, closing connection...")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

func updateLambdaConfig(cfg *config.Config, connectionID string) {
	ctx := context.TODO()
	lambdaName := cfg.GetLambdaFunctionName()
	log.Printf("DEBUG: Updating Lambda function '%s' with LAPTOP_ID=%s", lambdaName, connectionID)

	awsCfg, err := awssdkconfig.LoadDefaultConfig(ctx,
		awssdkconfig.WithSharedConfigProfile(cfg.AWSProfile),
		awssdkconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Printf("ERROR: Failed to load AWS config: %v", err)
		return
	}

	svc := lambda.NewFromConfig(awsCfg)

	log.Printf("DEBUG: Fetching current configuration for Lambda: %s", lambdaName)
	res, err := svc.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: &lambdaName,
	})
	if err != nil {
		log.Printf("ERROR: Failed to get Lambda config: %v", err)
		return
	}

	variables := res.Environment.Variables
	if variables == nil {
		variables = make(map[string]string)
	}
	variables["LAPTOP_ID"] = connectionID

	log.Println("DEBUG: Sending UpdateFunctionConfiguration request...")
	_, err = svc.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: &lambdaName,
		Environment: &lambdaTypes.Environment{
			Variables: variables,
		},
	})
	if err != nil {
		log.Printf("ERROR: Failed to update Lambda config: %v", err)
		return
	}

	log.Printf("SUCCESS: Lambda '%s' updated with LAPTOP_ID: %s", lambdaName, connectionID)
}

type AppMessage struct {
	Accion string      `json:"accion"`
	ID     interface{} `json:"id"`
	Body   interface{} `json:"body"`
}

type AppResponse struct {
	Accion string      `json:"accion"`
	ID     interface{} `json:"id"`
	Body   interface{} `json:"body,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func handleOffer(c *websocket.Conn, msg sigmsg.Msg) {
	log.Println("DEBUG: Starting WebRTC PeerConnection setup...")
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		log.Printf("ERROR: Failed to create PeerConnection: %v", err)
		return
	}

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("DEBUG: WebRTC Connection State Change: %s", s.String())
	})

	pc.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
		log.Printf("DEBUG: WebRTC ICE Gathering State Change: %s", s.String())
	})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("DEBUG: New DataChannel opened: %s", d.Label())
		d.OnOpen(func() {
			log.Printf("DEBUG: DataChannel '%s' is now OPEN", d.Label())
		})
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("DEBUG: Received DataChannel message: %s", string(msg.Data))

			var appMsg AppMessage
			if err := json.Unmarshal(msg.Data, &appMsg); err != nil {
				log.Printf("ERROR: Failed to unmarshal AppMessage: %v", err)
				return
			}

			log.Printf("DEBUG: Processing action: %s (ID: %v)", appMsg.Accion, appMsg.ID)

			var response AppResponse
			response.ID = appMsg.ID
			response.Accion = appMsg.Accion

			switch appMsg.Accion {
			case "get_stats":
				response.Body = map[string]interface{}{
					"status":  "running",
					"version": Version,
					"uptime":  "stable",
				}
			default:
				log.Printf("WARN: Action not recognized: %s", appMsg.Accion)
				response.Error = "action not recognized"
			}

			respPayload, _ := json.Marshal(response)
			if err := d.Send(respPayload); err != nil {
				log.Printf("ERROR: Failed to send response: %v", err)
			}
		})
	})

	offer := webrtc.SessionDescription{}
	data, _ := json.Marshal(msg.Data)
	if err := json.Unmarshal(data, &offer); err != nil {
		log.Printf("ERROR: Failed to unmarshal remote offer: %v", err)
		return
	}

	log.Println("DEBUG: Setting RemoteDescription...")
	if err := pc.SetRemoteDescription(offer); err != nil {
		log.Printf("ERROR: Failed to set remote description: %v", err)
		return
	}

	log.Println("DEBUG: Creating Answer...")
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("ERROR: Failed to create answer: %v", err)
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	log.Println("DEBUG: Setting LocalDescription...")
	if err := pc.SetLocalDescription(answer); err != nil {
		log.Printf("ERROR: Failed to set local description: %v", err)
		return
	}

	log.Println("DEBUG: Waiting for ICE gathering to complete...")
	<-gatherComplete
	log.Println("DEBUG: ICE gathering complete.")

	resp := sigmsg.Msg{
		Action: "sendSignal",
		To:     msg.From,
		Data:   *pc.LocalDescription(),
	}
	payload, _ := json.Marshal(resp)
	if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
		log.Printf("ERROR: Failed to send answer to signaling server: %v", err)
	} else {
		log.Printf("DEBUG: Successfully sent WebRTC answer to %s", msg.From)
	}
}
