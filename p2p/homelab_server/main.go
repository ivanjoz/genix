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

	"p2p_bridge/config"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var (
	installFlag   = flag.Bool("install", false, "Install the server as a systemd service and start it")
	uninstallFlag = flag.Bool("uninstall", false, "Uninstall the systemd service and remove the binary")
	Version       = "3.6.0-stable"
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
	if cfg.SignalingSocket == "" || cfg.ApiKey == "" {
		log.Fatal("ERROR: SIGNALING_SOCKET and API_KEY are required in credentials.json")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

		// 1. Build the correct WebSocket URL

		// Verified working path for AppSync Events: /event/realtime

		rtURL := cfg.SignalingSocket

		rtURL = strings.TrimSuffix(rtURL, "/")

		if !strings.HasSuffix(rtURL, "/event/realtime") {

			if strings.HasSuffix(rtURL, "/event") {

				rtURL += "/realtime"

			} else {

				rtURL += "/event/realtime"

			}

		}



	// 2. Prepare Authorization Header
	parsedAPIURL, _ := url.Parse(cfg.SignalingEndpoint)
	apiHost := parsedAPIURL.Host
	amzDate := time.Now().UTC().Format("20060102T150405Z")

	authHeader := map[string]string{
		"host":       apiHost,
		"x-api-key":  cfg.ApiKey,
		"x-amz-date": amzDate,
	}
	headerBytes, _ := json.Marshal(authHeader)
	
	// Base64URL (Raw/No padding) for the subprotocol
	headerB64Sub := base64.RawURLEncoding.EncodeToString(headerBytes)
	// Standard Base64 for the query parameter
	headerB64Query := base64.StdEncoding.EncodeToString(headerBytes)

	// 3. Connect using verified subprotocols
	dialer := websocket.Dialer{
		Subprotocols: []string{"aws-appsync-event-ws", "header-" + headerB64Sub},
	}

	finalURL := fmt.Sprintf("%s?header=%s&payload=e30=", rtURL, headerB64Query)

	log.Println("Version 4.1.0-flat-subscribe-structure")
	log.Printf("DEBUG: Connecting to AppSync Events at %s...", rtURL)
	c, resp, err := dialer.Dial(finalURL, nil)
	if err != nil {
		if resp != nil {
			log.Printf("ERROR: Handshake failed (Status %d)", resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			log.Printf("ERROR: Response Body: %s", string(body))
		}
		log.Fatalf("ERROR: Dial failed: %v", err)
	}
	defer c.Close()

	log.Printf("DEBUG: WebSocket established. Protocol: %s", resp.Header.Get("Sec-WebSocket-Protocol"))

	done := make(chan struct{})
	go func() {
		defer close(done)

		// A. Connection Initialization
		initMsg, _ := json.Marshal(AppSyncMessage{Type: "connection_init"})
		c.WriteMessage(websocket.TextMessage, initMsg)
		log.Println("DEBUG: Sent connection_init")

		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("DEBUG: WebSocket read error: %v (Type: %d)", err, messageType)
				return
			}

			log.Printf("DEBUG: Received: %s", string(message))

			var msg AppSyncMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "connection_ack":
				log.Println("DEBUG: AppSync Connection Acknowledged")
				
				// B. Subscribe to channel (Flattened structure as per working example)
				subID := fmt.Sprintf("sub-%d", time.Now().Unix())
				subMsg, _ := json.Marshal(map[string]interface{}{
					"id":            subID,
					"type":          "subscribe",
					"channel":       "genix-bridge/server",
					"authorization": authHeader,
				})
				c.WriteMessage(websocket.TextMessage, subMsg)
				log.Printf("DEBUG: Sent subscribe for 'genix-bridge/server' (ID: %s)", subID)

			case "subscribe_success":
				log.Printf("DEBUG: Subscribed successfully (ID: %s)", msg.ID)

			case "event":
				// Handle signal from AppSync Events
				payload, ok := msg.Payload.(map[string]interface{})
				if !ok { continue }

				events, ok := payload["events"].([]interface{})
				if !ok { continue }

				for _, event := range events {
					var signal AppSyncSignal
					eventBytes, _ := json.Marshal(event)
					// If event is a string, unmarshal from it
					var eventStr string
					if err := json.Unmarshal(eventBytes, &eventStr); err == nil {
						json.Unmarshal([]byte(eventStr), &signal)
					} else {
						json.Unmarshal(eventBytes, &signal)
					}

					if signal.Action == "offer" {
						log.Printf("DEBUG: Received WebRTC offer from %s", signal.From)
						handleOffer(cfg, signal)
					}
				}
			case "ka":
				// Keep alive
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

func publishSignal(cfg *config.Config, signal AppSyncSignal) error {
	// Use SignalingEndpoint directly for REST publishing
	publishURL := cfg.SignalingEndpoint
	if !strings.Contains(publishURL, "/event") {
		publishURL = strings.TrimSuffix(publishURL, "/") + "/event"
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"channel": "/genix-bridge/" + signal.To,
		"events":  []interface{}{signal},
	})

	req, _ := http.NewRequest("POST", publishURL, bytes.NewBuffer(reqBody))
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
		return fmt.Errorf("publish failed with status %d: %s", resp.StatusCode, string(body))
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
		From:   "genix-bridge",
		To:     signal.From,
		Action: "answer",
		Data:   string(answerJSON),
	}

	if err := publishSignal(cfg, resp); err != nil {
		log.Printf("ERROR: Failed to send answer signal: %v", err)
	} else {
		log.Printf("DEBUG: Successfully sent WebRTC answer to %s", signal.From)
	}
}
