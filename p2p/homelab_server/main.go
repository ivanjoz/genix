package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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
	"sync"
	"syscall"
	"time"

	"p2p_bridge/config"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var (
	installFlag        = flag.Bool("install", false, "Install the server as a systemd service and start it")
	uninstallFlag      = flag.Bool("uninstall", false, "Uninstall the systemd service and remove the binary")
	waitGatheringFlag  = flag.Bool("wait-gathering", false, "Wait for full ICE gathering before sending answer (slower but includes all candidates in SDP)")
	Version            = "4.2.0-fast-reload"
)

// Session management for Fast Reload
type SessionInfo struct {
	ClientID       string
	LastConnected  time.Time
	LastCandidates []webrtc.ICECandidateInit
}

var (
	activeSessions    = make(map[string]*SessionInfo)
	sessionMutex      sync.RWMutex
	globalCertificate *webrtc.Certificate
)

const (
	CertFile       = "webrtc_cert.json"
	SessionTimeout = 10 * time.Minute
)

// AppSync Realtime Messages
type AppSyncMessage struct {
	Type    string      `json:"type"`
	ID      string      `json:"id,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type AppSyncSignal struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Action       string `json:"action"`
	Data         string `json:"data"`
	SessionToken string `json:"sessionToken,omitempty"`
}

func initWebRTC() {
	// 1. Persistent Certificate
	cert, err := loadOrGenerateCert()
	if err != nil {
		log.Printf("ERROR: Failed to handle certificate: %v. Generating ephemeral.", err)
		// Try once more or fail
		cert, _ = loadOrGenerateCert()
	}
	globalCertificate = cert

	// 2. Session Cleanup
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			sessionMutex.Lock()
			for id, sess := range activeSessions {
				if time.Since(sess.LastConnected) > SessionTimeout {
					delete(activeSessions, id)
				}
			}
			sessionMutex.Unlock()
		}
	}()
}

func loadOrGenerateCert() (*webrtc.Certificate, error) {
	// Generate a new ECDSA key for the certificate
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return webrtc.GenerateCertificate(priv)
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

	initWebRTC()

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

			case "event", "data":
				// Handle signal from AppSync Events
				// The log shows: {"type":"data","event":"..."} or {"type":"event","payload":{"events":[...]}}
				
				var events []interface{}
				
				if msg.Type == "data" {
					// Handle the format: {"type":"data", "event":"..."}
					rawData, ok := msg.Payload.(map[string]interface{})
					if !ok {
						// In case msg.Payload is nil but event is at top level
						// AppSyncMessage might need to be more flexible or we unmarshal again
						var raw map[string]interface{}
						json.Unmarshal(message, &raw)
						if ev, ok := raw["event"]; ok {
							events = append(events, ev)
						}
					} else if ev, ok := rawData["event"]; ok {
						events = append(events, ev)
					} else {
						// Try unmarshaling the whole message again to find "event"
						var raw map[string]interface{}
						json.Unmarshal(message, &raw)
						if ev, ok := raw["event"]; ok {
							events = append(events, ev)
						}
					}
				} else {
					// Handle the format: {"type":"event","payload":{"events":[...]}}
					payload, ok := msg.Payload.(map[string]interface{})
					if ok {
						if evs, ok := payload["events"].([]interface{}); ok {
							events = evs
						}
					}
				}

				for _, event := range events {
					log.Printf("DEBUG: Processing event: %v", event)
					var signal AppSyncSignal
					eventBytes, _ := json.Marshal(event)
					// If event is a string, unmarshal from it
					var eventStr string
					if err := json.Unmarshal(eventBytes, &eventStr); err == nil {
						log.Printf("DEBUG: Unmarshaling signal from string: %s", eventStr)
						if err := json.Unmarshal([]byte(eventStr), &signal); err != nil {
							log.Printf("ERROR: Failed to unmarshal signal from string: %v", err)
							continue
						}
					} else {
						log.Printf("DEBUG: Unmarshaling signal from object")
						if err := json.Unmarshal(eventBytes, &signal); err != nil {
							log.Printf("ERROR: Failed to unmarshal signal from object: %v", err)
							continue
						}
					}

					log.Printf("DEBUG: Extracted Signal: From=%s, To=%s, Action=%s", signal.From, signal.To, signal.Action)

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

	eventData, _ := json.Marshal(signal)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"channel": "/genix-bridge/" + signal.To,
		"events":  []interface{}{string(eventData)},
	})

	log.Printf("DEBUG: Publishing to channel '%s' at URL %s", "/genix-bridge/"+signal.To, publishURL)
	log.Printf("DEBUG: Publish payload: %s", string(reqBody))

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

	log.Println("DEBUG: Signal published successfully")
	return nil
}

func handleOffer(cfg *config.Config, signal AppSyncSignal) {
	log.Printf("DEBUG: Starting WebRTC PeerConnection setup (SessionToken: %s)", signal.SessionToken)

	// Create a SettingEngine and enable IPv6 + Fixed Port Range
	s := webrtc.SettingEngine{}
	s.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP6,
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeTCP6,
		webrtc.NetworkTypeTCP4,
	})
	// Fix port range to help Full Cone NAT stability
	s.SetEphemeralUDPPortRange(50000, 50050)

	// Create the API object with the SettingEngine
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s))

	pc, err := api.NewPeerConnection(webrtc.Configuration{
		Certificates: []webrtc.Certificate{*globalCertificate},
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun.l.google.com:19305"}},
		},
	})
	if err != nil {
		log.Printf("ERROR: Failed to create PeerConnection: %v", err)
		return
	}

	// Session tracking
	var session *SessionInfo
	if signal.SessionToken != "" {
		sessionMutex.Lock()
		if sess, ok := activeSessions[signal.SessionToken]; ok {
			session = sess
			session.LastConnected = time.Now()
			session.LastCandidates = nil // Reset list for the new connection attempt
			log.Printf("DEBUG: Found active session for token %s (Resetting candidates)", signal.SessionToken)
		} else {
			session = &SessionInfo{
				ClientID:      signal.From,
				LastConnected: time.Now(),
			}
			activeSessions[signal.SessionToken] = session
			log.Printf("DEBUG: Created new session for token %s", signal.SessionToken)
		}
		sessionMutex.Unlock()
	}

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("DEBUG: WebRTC Connection State Change: %s", s.String())
		if s == webrtc.PeerConnectionStateConnected {
			if pc.SCTP() != nil && pc.SCTP().Transport() != nil && pc.SCTP().Transport().ICETransport() != nil {
				selectedPair, err := pc.SCTP().Transport().ICETransport().GetSelectedCandidatePair()
				if err == nil && selectedPair != nil {
					log.Printf("🎯 CONNECTION ESTABLISHED: %s <-> %s", selectedPair.Local.String(), selectedPair.Remote.String())
				}
			}
		}
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			candidateJSON := c.ToJSON()
			
			// Store in session for future fast reloads
			if session != nil {
				sessionMutex.Lock()
				session.LastCandidates = append(session.LastCandidates, candidateJSON)
				sessionMutex.Unlock()
			}

			if !*waitGatheringFlag {
				// Optimization: If already connected, don't waste signaling bandwidth.
				// We still stored it in the session above for the NEXT reload.
				if pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
					return
				}

				candidateData, _ := json.Marshal(candidateJSON)
				resp := AppSyncSignal{
					From:         "genix-bridge",
					To:           signal.From,
					Action:       "candidate",
					Data:         string(candidateData),
					SessionToken: signal.SessionToken,
				}
				publishSignal(cfg, resp)
			}
		}
	})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("DEBUG: New DataChannel opened: %s", d.Label())
		d.OnOpen(func() {
			log.Printf("DEBUG: DataChannel '%s' is now OPEN", d.Label())
			// Optional: Send a greeting when the channel opens
			d.SendText(fmt.Sprintf("Hello from Homelab Server! (Version %s)", Version))
		})
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			content := string(msg.Data)
			log.Printf("DEBUG: Received DataChannel message: %s", content)

			// Echo back or respond to specific messages
			response := map[string]interface{}{
				"type":      "response",
				"echo":      content,
				"timestamp": time.Now().UnixMilli(),
				"server":    "homelab",
			}
			respBytes, _ := json.Marshal(response)
			if err := d.Send(respBytes); err != nil {
				log.Printf("ERROR: Failed to send response to DataChannel: %v", err)
			} else {
				log.Printf("DEBUG: Sent response back to client")
			}
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

	var gatherComplete <-chan struct{}
	if *waitGatheringFlag {
		gatherComplete = webrtc.GatheringCompletePromise(pc)
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		log.Printf("ERROR: Failed to set local description: %v", err)
		return
	}

	if *waitGatheringFlag {
		log.Println("DEBUG: Waiting for full ICE gathering...")
		<-gatherComplete
	}

	answerJSON, _ := json.Marshal(pc.LocalDescription())
	resp := AppSyncSignal{
		From:         "genix-bridge",
		To:           signal.From,
		Action:       "answer",
		Data:         string(answerJSON),
		SessionToken: signal.SessionToken,
	}

	if err := publishSignal(cfg, resp); err != nil {
		log.Printf("ERROR: Failed to send answer signal: %v", err)
	} else {
		log.Printf("DEBUG: Successfully sent WebRTC answer (WaitGathering=%v) to %s", *waitGatheringFlag, signal.From)
	}
}
