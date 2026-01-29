package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"p2p_bridge/config"
)

var (
	installFlag   = flag.Bool("install", false, "Install the server as a systemd service and start it")
	uninstallFlag = flag.Bool("uninstall", false, "Uninstall the systemd service and remove the binary")
	Version       = "3.0.0-appsync"
)

// AppSync Realtime Messages
type AppSyncMessage struct {
	Type    string      `json:"type"`
	ID      string      `json:"id,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type AppSyncSignal struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"`
	Data   string `json:"data"`
}

func main() {
	flag.Parse()

	if *installFlag {
		if err := RunInstall(); err != nil {
			log.Fatalf("Installation failed: %v", err)
		}
		log.Println("Installation completed successfully!")
		return
	}

	if *uninstallFlag {
		if err := RunUninstall(); err != nil {
			log.Fatalf("Uninstallation failed: %v", err)
		}
		log.Println("Uninstallation completed successfully!")
		return
	}

	cfg := config.GetDefaultConfig()
	if cfg.SignalingEndpoint == "" || cfg.ApiKey == "" {
		log.Fatal("ERROR: SIGNALING_ENDPOINT and API_KEY are required in credentials.json")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Build Realtime WebSocket URL
	// https://xxx.appsync-api.region.amazonaws.com/graphql -> wss://xxx.appsync-realtime-api.region.amazonaws.com/graphql
	rtURL := strings.Replace(cfg.SignalingEndpoint, "https://", "wss://", 1)
	rtURL = strings.Replace(rtURL, "appsync-api", "appsync-realtime-api", 1)

	parsedURL, _ := url.Parse(cfg.SignalingEndpoint)
	host := parsedURL.Host

	header := map[string]string{
		"host":     host,
		"x-api-key": cfg.ApiKey,
	}
	headerBytes, _ := json.Marshal(header)
	headerB64 := base64.StdEncoding.EncodeToString(headerBytes)

	finalURL := fmt.Sprintf("%s?header=%s&payload=e30=", rtURL, headerB64)

	log.Printf("DEBUG: Connecting to AppSync Realtime at %s...", rtURL)
	c, _, err := websocket.DefaultDialer.Dial(finalURL, http.Header{"Sec-WebSocket-Protocol": []string{"graphql-ws"}})
	if err != nil {
		log.Fatalf("ERROR: Dial failed: %v", err)
	}
	defer c.Close()

	// 1. Connection Init
	initMsg, _ := json.Marshal(AppSyncMessage{Type: "connection_init"})
	c.WriteMessage(websocket.TextMessage, initMsg)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("DEBUG: WebSocket read error:", err)
				return
			}

			var msg AppSyncMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "connection_ack":
				log.Println("DEBUG: AppSync Connection Acknowledged")
				// 2. Subscribe to signals
				subID := "sub-1"
				subPayload := map[string]interface{}{
					"data": "subscription onSignal($to: String!) { onSignal(to: $to) { from to action data } }",
					"variables": map[string]string{
						"to": "laptop",
					},
				}
				// AppSync sub requires 'data' (query) and 'extensions' (auth)
				startMsg, _ := json.Marshal(map[string]interface{}{
					"type": "start",
					"id":   subID,
					"payload": map[string]interface{}{
						"data": subPayload["data"],
						"variables": subPayload["variables"],
						"extensions": map[string]interface{}{
							"authorization": header,
						},
					},
				})
				c.WriteMessage(websocket.TextMessage, startMsg)
				log.Println("DEBUG: Subscription request sent for 'laptop'")

			case "data":
				// Handle signal
				payload := msg.Payload.(map[string]interface{})
				data := payload["data"].(map[string]interface{})
				onSignal := data["onSignal"].(map[string]interface{})

				signal := AppSyncSignal{
					From:   onSignal["from"].(string),
					To:     onSignal["to"].(string),
					Action: onSignal["action"].(string),
					Data:   onSignal["data"].(string),
				}

				if signal.Action == "offer" {
					log.Printf("DEBUG: Received WebRTC offer from %s", signal.From)
					handleOffer(cfg, signal)
				}
			case "ka":
				// Keep alive, ignore
			case "error":
				log.Printf("ERROR: AppSync Error: %v", msg.Payload)
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("DEBUG: Interrupt signal received, closing...")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

func sendMutation(cfg *config.Config, signal AppSyncSignal) error {
	query := `mutation sendSignal($from: String!, $to: String!, $action: String!, $data: String!) {
		sendSignal(from: $from, to: $to, action: $action, data: $data) {
			from to action data
		}
	}`

	variables := map[string]string{
		"from":   signal.From,
		"to":     signal.To,
		"action": signal.Action,
		"data":   signal.Data,
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})

	req, _ := http.NewRequest("POST", cfg.SignalingEndpoint, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.ApiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mutation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func handleOffer(cfg *config.Config, signal AppSyncSignal) {
	log.Println("DEBUG: Starting WebRTC PeerConnection setup with IPv6 optimization...")

	// Create a SettingEngine and enable IPv6
	s := webrtc.SettingEngine{}
	s.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP6,
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeTCP6,
		webrtc.NetworkTypeTCP4,
	})

	// Create the API object with the SettingEngine
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s))

	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun.l.google.com:19305"}}, // Multiple STUNs for better gathering
		},
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

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			candidateStr := c.String()
			log.Printf("DEBUG: Local ICE Candidate gathered: %s", candidateStr)
			if strings.Contains(candidateStr, ":") && !strings.Contains(candidateStr, ".") {
				log.Printf("DEBUG: Found IPv6 Candidate")
			}
		}
	})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("DEBUG: New DataChannel opened: %s", d.Label())
		d.OnOpen(func() {
			log.Printf("DEBUG: DataChannel '%s' is now OPEN", d.Label())
		})
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("DEBUG: Received DataChannel message: %s", string(msg.Data))
			// Business logic here
		})
	})

	offer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(signal.Data), &offer); err != nil {
		log.Printf("ERROR: Failed to unmarshal remote offer: %v", err)
		return
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		log.Printf("ERROR: Failed to set remote description: %v", err)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("ERROR: Failed to create answer: %v", err)
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		log.Printf("ERROR: Failed to set local description: %v", err)
		return
	}

	<-gatherComplete

	answerJSON, _ := json.Marshal(pc.LocalDescription())
	resp := AppSyncSignal{
		From:   "laptop",
		To:     signal.From,
		Action: "answer",
		Data:   string(answerJSON),
	}

	if err := sendMutation(cfg, resp); err != nil {
		log.Printf("ERROR: Failed to send answer mutation: %v", err)
	} else {
		log.Printf("DEBUG: Successfully sent WebRTC answer to %s", signal.From)
	}
}
