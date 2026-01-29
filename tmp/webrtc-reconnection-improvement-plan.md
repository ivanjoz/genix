# WebRTC Reconnection Improvement Plan

## Executive Summary

This document outlines a strategy to implement two-mode WebRTC reconnection: **Fast Reload Connection** (optimized for quick reconnections) and **Normal Connection** (reliable fallback). The plan addresses the challenge of reconnecting with a full cone NAT and dynamic IP server environment.

## Problem Statement

**Current Situation:**
- Server has dynamic IP (no fixed IP)
- Network: Full cone NAT with hairpin capability
- STUN result: `0x000002` (Independent Mapping, Independent Filter, random port)
- Reconnections take ~1-2 seconds due to full ICE gathering cycle
- No candidate caching or session persistence

**Goal:**
- Reduce reconnection time to <500ms for fast reload
- Maintain reliability with normal connection fallback
- Handle dynamic IP changes gracefully

## Network Environment Analysis

### Server Side (Full Cone NAT, Dynamic IP)
```
STUN Result: 0x000002
- Mapping Type: Independent Mapping (endpoint-independent)
- Filtering Type: Independent Filter (endpoint-independent)
- Port Allocation: Random
- Hairpinning: Supported

Implications:
- Server's public IP:port can change (dynamic IP)
- However, NAT behavior is consistent (endpoint-independent)
- Direct connections are possible without additional relay
- Server's ICE candidates change when IP changes, but network behavior is stable
```

### Client Side
- Browser-based with SimplePeer
- Can cache candidates in localStorage
- Can detect connection failures and trigger fallback

### IPv6 Optimization Opportunity
**Critical Insight:** IPv6 can dramatically improve fast reload connection speed

**Why IPv6 is Better for Fast Reload:**
- No NAT traversal required (end-to-end connectivity)
- Addresses are more stable (especially with IPv6 stable prefixes)
- No STUN discovery needed for public IP
- Lower latency due to fewer network hops
- Direct connection without intermediate NAT devices

**Current Implementation Status:**
- Server already prioritizes IPv6: `webrtc.NetworkTypeUDP6`, `webrtc.NetworkTypeTCP6`
- Client can handle IPv6 candidates
- Both sides support dual-stack operation

**Fast Reload Strategy with IPv6:**
```
Priority Order for Fast Reload:
1. IPv6 Host Candidates (most stable, fastest)
   - Client's global IPv6 address → Server's global IPv6 address
   - Typically direct connection, no NAT
   - Addresses rarely change compared to IPv4 behind NAT
   
2. IPv6 srflx Candidates
   - If host IPv6 fails, try STUN-reflexive IPv6
   - Still usually faster than IPv4
   
3. IPv4 srflx Candidates (fallback)
   - Behind full cone NAT, but public IP may have changed
   - Less reliable than IPv6
   
4. Normal Mode (full discovery)
   - If all fast reload candidates fail
```

**IPv6-Specific Optimizations:**
- Cache IPv6 host candidates separately from IPv4
- Prioritize IPv6 candidates during fast reload injection
- Track IPv6 connection success rate separately
- If IPv6 available, prefer it even if IPv4 cache is fresher
- Log IPv6 vs IPv4 connection times independently

**Implementation Notes:**
- Client should check for IPv6 support before attempting
- Server should prioritize IPv6 in candidate gathering
- Cache structure should separate IPv6 and IPv4 candidates
- Fallback chain: IPv6 fast → IPv4 fast → Normal mode
- Expected IPv6 fast reload time: <300ms (even faster than IPv4)

## Two-Mode Connection Strategy

### Mode 1: Fast Reload Connection
**Purpose:** Quick reconnection when server network hasn't changed significantly

**When to use:**
- Page reload within 5 minutes of last successful connection
- No network configuration changes detected
- Cached server credentials available

**Optimizations:**
1. Pre-inject cached server ICE candidates before signaling
2. Skip ICE candidate gathering on server if possible
3. Use cached DTLS fingerprint validation
4. Trickle ICE immediately without waiting
5. Prioritize previously successful ICE candidate types

**Expected Time:**
- IPv6 connections: <300ms (even faster - no NAT traversal)
- IPv4 connections: <500ms

### Mode 2: Normal Connection
**Purpose:** Reliable connection with full discovery

**When to use:**
- First connection ever
- Fast reload mode fails or times out
- Cached credentials are stale (>5 minutes)
- Network changes detected

**Behavior:**
- Full ICE gathering cycle
- Complete candidate exchange via signaling
- Standard WebRTC flow

**Expected Time:** 1-2 seconds

## Server-Side Changes (main.go)

### 1. DTLS Certificate Reuse
**Rationale:** Certificate generation takes 50-200ms and causes unnecessary delays

**Implementation:**
```go
// Global variable to store certificate
var globalCertificate *webrtc.Certificate

// Initialize certificate on startup
func initGlobalCertificate() error {
    cert, err := webrtc.GenerateCertificate()
    if err != nil {
        return err
    }
    globalCertificate = cert
    return nil
}

// Use certificate in PeerConnection
api := webrtc.NewAPI(
    webrtc.WithSettingEngine(s),
    webrtc.WithCertificate(globalCertificate),
)
```

**Considerations:**
- Certificate must be regenerated on server restart
- Implement certificate rotation mechanism for security (monthly)
- Log certificate initialization

### 2. Session Persistence (Reconnection Token)
**Rationale:** Enable server to recognize returning clients and optimize response

**Implementation:**
```go
type SessionInfo struct {
    ClientID      string
    LastConnected time.Time
    LastIP        string
    PreferredICE  string // Last successful candidate type
    SessionToken  string // UUID for this session
}

var activeSessions = sync.Map{} // map[clientID]SessionInfo

// In handleOffer:
if signal.SessionToken != "" {
    if session, ok := activeSessions.Load(signal.SessionToken); ok {
        // Fast path: returning client
        log.Printf("Fast reload: Recognized client %s", signal.SessionToken)
        // Use optimized ICE gathering
    }
}
```

**Considerations:**
- Session TTL: 5 minutes for fast reload eligibility
- Clean up expired sessions periodically
- Don't store sensitive data in sessions

### 3. Optimized ICE Gathering for Fast Reload
**Rationale:** Skip unnecessary gathering when client already has candidates

**Implementation:**
```go
func handleOffer(cfg *config.Config, signal AppSyncSignal) {
    isFastReload := false
    var cachedCandidates []webrtc.ICECandidateInit
    
    // Check if this is a fast reload attempt
    if signal.SessionToken != "" {
        if session, ok := activeSessions.Load(signal.SessionToken); ok {
            sess := session.(SessionInfo)
            if time.Since(sess.LastConnected) < 5*time.Minute {
                isFastReload = true
                cachedCandidates = sess.LastCandidates // From session cache
            }
        }
    }
    
    // Create PeerConnection
    pc, err := api.NewPeerConnection(webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {URLs: []string{"stun:stun.l.google.com:19302"}},
        },
    })
    
    if isFastReload && len(cachedCandidates) > 0 {
        // Fast path: Immediately add cached candidates to answer
        log.Printf("Using cached candidates for fast reload")
        for _, c := range cachedCandidates {
            pc.AddICECandidate(c)
        }
        // Send answer immediately with cached candidates
        go sendFastAnswer(pc, cfg, signal)
    } else {
        // Normal path: Wait for ICE gathering
        go sendNormalAnswer(pc, cfg, signal)
    }
}
```

**Considerations:**
- Cached candidates might be stale (server IP changed)
- Add validation to check if cached candidates still work
- Fall back to normal gathering if fast path fails

### 4. Enhanced Candidate Logging
**Rationale:** Better visibility into connection paths and failures

**Implementation:**
```go
pc.OnICECandidate(func(c *webrtc.ICECandidate) {
    if c != nil {
        cType := "unknown"
        if c.Type != nil {
            cType = c.Type.String()
        }
        
        // Log with metadata
        log.Printf("[ICE] Candidate: Type=%s, Protocol=%s, Address=%s:%d",
            cType, c.Protocol, c.Address, c.Port)
        
        // Store in session if applicable
        if signal.SessionToken != "" {
            if session, ok := activeSessions.Load(signal.SessionToken); ok {
                sess := session.(SessionInfo)
                sess.LastCandidates = append(sess.LastCandidates, c.ToJSON())
                activeSessions.Store(signal.SessionToken, sess)
            }
        }
    }
})
```

### 5. Connection Type Detection
**Rationale:** Log which ICE pair succeeded for future optimization

**Implementation:**
```go
pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
    if s == webrtc.PeerConnectionStateConnected {
        // Get selected ICE pair
        if pair, err := pc.SCTP().Transport().ICETransport().GetSelectedCandidatePair(); err == nil {
            log.Printf("[SUCCESS] %s connection established via %s",
                getMode(signal.SessionToken), // "FastReload" or "Normal"
                getPairDescription(pair))
            
            // Update session with successful candidate type
            if signal.SessionToken != "" {
                updateSessionWithSuccess(signal.SessionToken, pair)
            }
        }
    }
})
```

## Client-Side Changes (wss-webrtc.ts)

### 1. Candidate Caching System
**Rationale:** Store server candidates for fast reload injection

**Implementation:**
```typescript
interface CachedConnectionData {
  ipv6HostCandidates: RTCIceCandidateInit[];
  ipv6SrflxCandidates: RTCIceCandidateInit[];
  ipv4SrflxCandidates: RTCIceCandidateInit[];
  ipv4RelayCandidates: RTCIceCandidateInit[];
  lastSuccessfulProtocol: 'ipv6' | 'ipv4' | 'both';
  timestamp: number;
  sessionId: string;
  lastConnectedIPv6?: boolean;
}

class ConnectionCache {
  private static readonly CACHE_KEY = 'webrtc_connection_cache';
  private static readonly MAX_AGE = 5 * 60 * 1000; // 5 minutes

  static save(data: CachedConnectionData): void {
    localStorage.setItem(this.CACHE_KEY, JSON.stringify(data));
    console.log('[Cache] Connection data saved');
  }

  static load(): CachedConnectionData | null {
    const cached = localStorage.getItem(this.CACHE_KEY);
    if (!cached) return null;

    const data: CachedConnectionData = JSON.parse(cached);
    
    // Check if cache is still valid
    if (Date.now() - data.timestamp > this.MAX_AGE) {
      console.log('[Cache] Data expired, ignoring');
      return null;
    }

    console.log('[Cache] Loaded valid cached data');
    return data;
  }

  static clear(): void {
    localStorage.removeItem(this.CACHE_KEY);
    console.log('[Cache] Cleared');
  }
}
```

**Considerations:**
- Cache expiration prevents stale data usage
- Browser localStorage is per-origin and persistent
- Clear cache on connection failures

### 2. Two-Mode Connection Logic
**Rationale:** Implement fast reload with automatic fallback

**Implementation:**
```typescript
export class WSSWebRTC {
  private connectionMode: 'fast' | 'normal' = 'fast';
  private connectionTimeout: NodeJS.Timeout | null = null;
  private readonly FAST_MODE_TIMEOUT = 2000; // 2 seconds

  async connect(): Promise<void> {
    // Try fast reload first
    const cached = ConnectionCache.load();
    
    if (cached) {
      console.log('[Mode] Attempting FAST RELOAD connection');
      this.connectionMode = 'fast';
      await this.connectFastReload(cached);
      
      // Set timeout for fallback
      this.connectionTimeout = setTimeout(() => {
        if (!this.peer?.connected) {
          console.log('[Mode] Fast reload timeout, switching to NORMAL');
          this.switchToNormalMode();
        }
      }, this.FAST_MODE_TIMEOUT);
    } else {
      console.log('[Mode] No cache available, using NORMAL connection');
      this.connectionMode = 'normal';
      await this.connectNormal();
    }
  }

  private async connectFastReload(cached: CachedConnectionData): Promise<void> {
    // Create SimplePeer with cached candidates
    this.peer = new SimplePeer({
      initiator: true,
      trickle: false, // Disable trickle for fast reload
      config: { 
        iceServers: this.config.stunServers.map(urls => ({ urls }))
      }
    });

    // Pre-inject cached server candidates with IPv6 prioritization
    // Priority: IPv6 Host → IPv6 srflx → IPv4 srflx → IPv4 relay
    console.log('[FastReload] Injecting candidates with IPv6 priority...');
    
    let candidatesInjected = 0;
    
    // 1. IPv6 Host Candidates (most stable, fastest)
    if (cached.ipv6HostCandidates && cached.ipv6HostCandidates.length > 0) {
      console.log(`[FastReload] Injecting ${cached.ipv6HostCandidates.length} IPv6 host candidates`);
      cached.ipv6HostCandidates.forEach(candidate => {
        this.peer.addIceCandidate(candidate);
        candidatesInjected++;
      });
    }
    
    // 2. IPv6 srflx Candidates
    if (cached.ipv6SrflxCandidates && cached.ipv6SrflxCandidates.length > 0) {
      console.log(`[FastReload] Injecting ${cached.ipv6SrflxCandidates.length} IPv6 srflx candidates`);
      cached.ipv6SrflxCandidates.forEach(candidate => {
        this.peer.addIceCandidate(candidate);
        candidatesInjected++;
      });
    }
    
    // 3. IPv4 srflx Candidates (fallback)
    if (cached.ipv4SrflxCandidates && cached.ipv4SrflxCandidates.length > 0) {
      console.log(`[FastReload] Injecting ${cached.ipv4SrflxCandidates.length} IPv4 srflx candidates`);
      cached.ipv4SrflxCandidates.forEach(candidate => {
        this.peer.addIceCandidate(candidate);
        candidatesInjected++;
      });
    }
    
    // 4. IPv4 Relay Candidates (last resort)
    if (cached.ipv4RelayCandidates && cached.ipv4RelayCandidates.length > 0) {
      console.log(`[FastReload] Injecting ${cached.ipv4RelayCandidates.length} IPv4 relay candidates`);
      cached.ipv4RelayCandidates.forEach(candidate => {
        this.peer.addIceCandidate(candidate);
        candidatesInjected++;
      });
    }
    
    console.log(`[FastReload] Total candidates injected: ${candidatesInjected}`);
    console.log(`[FastReload] Last successful protocol: ${cached.lastSuccessfulProtocol}`);

    // Set up event handlers
    this.setupPeerHandlers();

    // Send offer immediately
    this.peer.once('signal', (offer: any) => {
      // Add session token to signal
      offer.sessionToken = cached.sessionId;
      this.sendSignal('offer', JSON.stringify(offer));
    });
  }

  private async connectNormal(): Promise<void> {
    // Standard connection flow
    this.peer = new SimplePeer({
      initiator: true,
      trickle: this.config.trickle,
      config: { 
        iceServers: this.config.stunServers.map(urls => ({ urls }))
      }
    });

    this.setupPeerHandlers();
  }

  private switchToNormalMode(): void {
    console.log('[Fallback] Switching to normal connection mode');
    
    // Clear timeout
    if (this.connectionTimeout) {
      clearTimeout(this.connectionTimeout);
      this.connectionTimeout = null;
    }

    // Clear stale cache
    ConnectionCache.clear();

    // Destroy old peer
    if (this.peer) {
      this.peer.destroy();
    }

    // Start normal connection
    this.connectionMode = 'normal';
    this.connectNormal();
  }
}
```

**Considerations:**
- Fast mode timeout should be long enough for candidate exchange
- Don't cache failed connections
- Provide visual feedback to user about connection mode

### 3. Candidate Storage on Success
**Rationale:** Save successful candidates for future fast reloads

**Implementation:**
```typescript
private setupPeerHandlers(): void {
  this.peer.on('connect', () => {
    console.log(`[WSSWebRTC] P2P Connected in ${this.connectionMode} mode!`);
    this.emit('connect');

    // Cache successful connection data
    if (this.peer && this.peer.peerConnection) {
      const pc: RTCPeerConnection = this.peer.peerConnection;
      
      // Get selected ICE pair
      pc.getStats().then(stats => {
        // Find transport stats to get selected pair
        let selectedPair: any = null;
        
        stats.forEach(report => {
          if (report.type === 'transport') {
            selectedPair = report;
          }
        });

        // Determine connection type from selected pair
        const connType = selectedPair?.selectedCandidatePairId || 'unknown';
        
        // Get local description to extract server candidates
        const remoteDesc = pc.remoteDescription;
        if (remoteDesc) {
          const candidates = this.extractCandidatesFromSDP(remoteDesc.sdp);
          
          ConnectionCache.save({
            ipv6HostCandidates: candidates.ipv6Host,
            ipv6SrflxCandidates: candidates.ipv6Srflx,
            ipv4SrflxCandidates: candidates.ipv4Srflx,
            ipv4RelayCandidates: candidates.ipv4Relay,
            lastSuccessfulProtocol: candidates.lastSuccessfulProtocol,
            timestamp: Date.now(),
            sessionId: this.generateSessionToken(),
            lastConnectedIPv6: candidates.lastConnectedIPv6
          });
        }
      });
    }
  });

  this.peer.on('error', (err: any) => {
    console.error('[WSSWebRTC] Error:', err);
    
    // If fast mode fails, switch to normal
    if (this.connectionMode === 'fast') {
      this.switchToNormalMode();
    } else {
      this.emit('error', err);
    }
  });
}
```

### 4. SDP Candidate Extraction
**Rationale:** Parse server candidates from remote description

**Implementation:**
```typescript
private extractCandidatesFromSDP(sdp: string | null): {
  ipv6Host: RTCIceCandidateInit[];
  ipv6Srflx: RTCIceCandidateInit[];
  ipv4Srflx: RTCIceCandidateInit[];
  ipv4Relay: RTCIceCandidateInit[];
  lastSuccessfulProtocol: 'ipv6' | 'ipv4' | 'both';
  lastConnectedIPv6: boolean;
} {
  if (!sdp) {
    return {
      ipv6Host: [],
      ipv6Srflx: [],
      ipv4Srflx: [],
      ipv4Relay: [],
      lastSuccessfulProtocol: 'both',
      lastConnectedIPv6: false
    };
  }

  const result = {
    ipv6Host: [] as RTCIceCandidateInit[],
    ipv6Srflx: [] as RTCIceCandidateInit[],
    ipv4Srflx: [] as RTCIceCandidateInit[],
    ipv4Relay: [] as RTCIceCandidateInit[],
    lastSuccessfulProtocol: 'both' as 'ipv6' | 'ipv4' | 'both',
    lastConnectedIPv6: false
  };

  const lines = sdp.split('\n');
  let hasIPv6 = false;
  let hasIPv4 = false;

  for (const line of lines) {
    if (line.startsWith('a=candidate:')) {
      // Parse candidate line
      // Format: a=candidate:<foundation> <component> <transport> <priority> <address> <port> <typ> <type>
      const parts = line.substring(12).split(' ');
      if (parts.length >= 8) {
        const address = parts[4];
        const candidateType = parts[7]; // 'host', 'srflx', 'prflx', 'relay'
        
        // Detect IP version
        const isIPv6 = address.includes(':') && address !== '::';
        isIPv6 ? hasIPv6 = true : hasIPv4 = true;

        const candidateInit: RTCIceCandidateInit = {
          candidate: line.substring(2), // Remove 'a='
          sdpMid: '0',
          sdpMLineIndex: 0
        };

        // Categorize by IP version and type
        if (isIPv6) {
          if (candidateType === 'host') {
            result.ipv6Host.push(candidateInit);
          } else if (candidateType === 'srflx') {
            result.ipv6Srflx.push(candidateInit);
          }
        } else {
          if (candidateType === 'srflx') {
            result.ipv4Srflx.push(candidateInit);
          } else if (candidateType === 'relay') {
            result.ipv4Relay.push(candidateInit);
          }
        }
      }
    }
  }

  // Determine protocol success
  if (hasIPv6 && hasIPv4) {
    result.lastSuccessfulProtocol = 'both';
  } else if (hasIPv6) {
    result.lastSuccessfulProtocol = 'ipv6';
  } else if (hasIPv4) {
    result.lastSuccessfulProtocol = 'ipv4';
  }
  
  result.lastConnectedIPv6 = hasIPv6;

  console.log('[CandidateExtraction] IPv6 Host:', result.ipv6Host.length, 
              'IPv6 srflx:', result.ipv6Srflx.length,
              'IPv4 srflx:', result.ipv4Srflx.length,
              'IPv4 relay:', result.ipv4Relay.length,
              'Protocol:', result.lastSuccessfulProtocol);

  return result;
}
```

### 5. Session Token Generation
**Rationale:** Unique identifier for connection sessions

**Implementation:**
```typescript
private generateSessionToken(): string {
  // Generate UUID-like token
  return 'session-' + Date.now().toString(36) + '-' + 
         Math.random().toString(36).substring(2, 9);
}
```

## Connection Flows

### Fast Reload Connection Flow

```
1. Client Page Reload
   ↓
2. Check localStorage for cached connection data
   ↓
3. If cache exists and is fresh (<5 min):
   ↓
4. Create SimplePeer with trickle: false
   ↓
5. Pre-inject cached server candidates with IPv6 priority:
   - IPv6 host candidates (most stable)
   - IPv6 srflx candidates
   - IPv4 srflx candidates (fallback)
   - IPv4 relay candidates (last resort)
   ↓
6. Send offer with session token
   ↓
7. Server recognizes session token
   ↓
8. Server creates PeerConnection with reused DTLS cert
   ↓
9. Server immediately answers with cached candidates
   ↓
10. ICE negotiation completes quickly
   ↓
11. Connection established:
      - IPv6: <300ms (even faster!)
      - IPv4: <500ms

Fallback Conditions:
- Cache missing or expired
- Server doesn't recognize session
- ICE connection fails
- Timeout (>2s)
```

### Normal Connection Flow

```
1. Client connects (or fast reload failed)
   ↓
2. Create SimplePeer with trickle: true
   ↓
3. Send offer (with or without session token)
   ↓
4. Server creates PeerConnection
   ↓
5. Server gathers ICE candidates
   ↓
6. Server sends answer
   ↓
7. Server trickles candidates to client
   ↓
8. Client trickles candidates to server
   ↓
9. ICE negotiation completes
   ↓
10. Connection established (1-2s)
   ↓
11. Cache successful candidates for next fast reload
```

### Fallback Mechanism

```
Fast Reload Mode
    ↓
   [Timeout: 2s]
    ↓
   [Check: Connected?]
    ↓
   NO → Destroy peer
    ↓
   Clear cache
    ↓
   Switch to Normal Mode
    ↓
   [Normal Connection Flow]
    ↓
   YES → Save to cache
```

## Implementation Steps

### Phase 1: Server Infrastructure (main.go)

**Step 1: Add DTLS certificate reuse**
- [ ] Create global certificate variable
- [ ] Initialize certificate in main()
- [ ] Configure PeerConnection API with certificate
- [ ] Add certificate logging

**Step 2: Implement session management**
- [ ] Define SessionInfo struct
- [ ] Create activeSessions sync.Map
- [ ] Add session cleanup goroutine (every minute)
- [ ] Add session token parsing from signals

**Step 3: Add fast path in handleOffer**
- [ ] Check for session token
- [ ] Validate session age
- [ ] Retrieve cached candidates
- [ ] Add candidates immediately if valid

**Step 4: Enhance logging**
- [ ] Log connection mode (fast/normal)
- [ ] Log selected ICE pair details
- [ ] Log candidate gathering progress
- [ ] Add connection timing metrics

**Step 5: Add signaling improvements**
- [ ] Pass session token in offers
- [ ] Include connection type in answers
- [ ] Add fast mode indicator in logs

### Phase 2: Client Infrastructure (wss-webrtc.ts)

**Step 1: Implement ConnectionCache class**
- [ ] Create save/load/clear methods
- [ ] Add cache expiration logic
- [ ] Add cache validation
- [ ] Add cache logging

**Step 2: Add two-mode connection logic**
- [ ] Implement connectFastReload()
- [ ] Implement connectNormal()
- [ ] Add switchToNormalMode()
- [ ] Add timeout mechanism
- [ ] Implement IPv6-first candidate injection in connectFastReload()
- [ ] Add IPv6 priority logging (host → srflx → IPv4)
- [ ] Track last successful protocol (IPv6/IPv4/both)

**Step 3: Implement candidate extraction**
- [ ] Parse SDP for candidates
- [ ] Store candidate metadata
- [ ] Handle multiple candidates
- [ ] Categorize candidates by IP version (IPv6/IPv4)
- [ ] Separate candidates by type (host/srflx/relay)
- [ ] Store IPv6 and IPv4 candidates separately in cache
- [ ] Detect IPv6 addresses in candidate strings
- [ ] Extract candidate type (host/srflx/prflx/relay) from SDP

**Step 4: Update event handlers**
- [ ] Save cache on successful connect
- [ ] Handle fast mode errors
- [ ] Implement automatic fallback
- [ ] Add mode logging
- [ ] Log IPv6 vs IPv4 connection success
- [ ] Track and log connection times by protocol
- [ ] Save IPv6/IPv4 separation info to cache

**Step 5: Add session management**
- [ ] Generate session tokens
- [ ] Pass token in offers
- [ ] Track connection mode

### Phase 3: Integration and Testing

**Step 1: Integrate changes**
- [ ] Update AppSync message format to include session token
- [ ] Test both connection modes
- [ ] Verify fallback mechanism

**Step 2: Add monitoring**
- [ ] Track fast vs normal connection counts
- [ ] Measure connection times
- [ ] Log fallback reasons

**Step 3: Performance testing**
- [ ] Measure fast reload connection time
- [ ] Measure normal connection time
- [ ] Test cache expiration
- [ ] Test fallback scenarios
- [ ] Compare IPv6 vs IPv4 connection times
- [ ] Measure IPv6 fast reload performance (target <300ms)
- [ ] Measure IPv4 fast reload performance (target <500ms)
- [ ] Test IPv6-first candidate prioritization
- [ ] Verify IPv6 to IPv6 direct connections
- [ ] Test IPv6 fallback to IPv4

## Testing Strategy

### Unit Tests

**Server Tests:**
- [ ] Certificate initialization and reuse
- [ ] Session creation and expiration
- [ ] Fast path candidate injection
- [ ] Session cleanup

**Client Tests:**
- [ ] Cache save/load operations
- [ ] Cache expiration logic
- [ ] Candidate extraction from SDP
- [ ] Mode switching logic

### Integration Tests

**Scenario 1: First Connection (Normal Mode)**
```
1. Clear all caches
2. Connect from client
3. Verify: Normal mode is used
4. Verify: Server generates fresh candidates
5. Verify: Connection succeeds
6. Verify: Client caches candidates
```

**Scenario 2: Fast Reload (Within Cache TTL)**
```
1. Establish successful connection
2. Reload page immediately
3. Verify: Fast mode is attempted
4. Verify: Cached candidates are pre-injected
5. Verify: Connection time <500ms
6. Verify: Server recognizes session token
```

**Scenario 3: Cache Expiration**
```
1. Establish successful connection
2. Wait 6 minutes (cache expires)
3. Reload page
4. Verify: Cache is ignored
5. Verify: Normal mode is used
6. Verify: Connection succeeds
```

**Scenario 4: Fallback on Failure**
```
1. Establish successful connection
2. Reload page
3. Manually block ICE connection
4. Wait for 2s timeout
5. Verify: Fallback to normal mode triggers
6. Verify: Stale cache is cleared
7. Verify: Connection succeeds in normal mode
```

**Scenario 5: Server IP Change**
```
1. Establish successful connection
2. Change server IP (simulate)
3. Reload page
4. Verify: Fast mode is attempted
5. Verify: Fast mode fails (stale candidates)
6. Verify: Fallback to normal mode
7. Verify: New candidates are gathered
8. Verify: New cache is saved
```

**Scenario 6: IPv6 Fast Reload with IPv6 Prioritization**
```
1. Ensure both server and client have IPv6 connectivity
2. Establish successful connection
3. Verify: IPv6 host candidates are cached
4. Reload page within cache TTL
5. Verify: Fast mode is attempted
6. Verify: IPv6 candidates are injected first (host → srflx)
7. Verify: IPv6 to IPv6 direct connection succeeds
8. Verify: Connection time <300ms (IPv6 target)
9. Verify: Server logs show IPv6 priority order
10. Verify: Last successful protocol is logged as 'ipv6' or 'both'
```

### Performance Benchmarks

**Metrics to Track:**
- [ ] Average connection time (fast mode)
- [ ] Average connection time (normal mode)
- [ ] Fast mode success rate
- [ ] Fallback rate
- [ ] Cache hit rate

**Target Metrics:**
- IPv6 fast reload connection time: <300ms
- IPv4 fast reload connection time: <500ms
- Normal mode connection time: <2s
- Fast mode success rate: >80%
- IPv6 connection success rate: >70% (when IPv6 available)
- Fallback rate: <20%

## Error Handling and Edge Cases

### Edge Cases to Handle

1. **Cache Corruption**
   - Validate JSON parsing
   - Handle malformed data
   - Clear corrupt cache

2. **Multiple Concurrent Connections**
   - Use unique session tokens per connection
   - Don't share cache between connections

3. **Network Changes During Connection**
   - Detect network changes
   - Invalidate cache on network change
   - Trigger reconnection

4. **Server Restart**
   - Server certificate changes
   - All sessions become invalid
   - Clients fall back to normal mode

5. **Browser Storage Quota**
   - localStorage may be full
   - Gracefully handle quota errors
   - Fall back to normal mode

### Error Recovery

```typescript
// Client error handling
private handleConnectionError(err: any): void {
  console.error('[WSSWebRTC] Connection error:', err);
  
  // Clear potentially corrupt cache
  if (this.connectionMode === 'fast') {
    ConnectionCache.clear();
  }
  
  // Determine recovery strategy
  if (err.code === 'ERR_ICE_CONNECTION_FAILED') {
    if (this.connectionMode === 'fast') {
      console.log('[Recovery] Fast mode failed, retrying with normal mode');
      this.switchToNormalMode();
      return;
    }
  }
  
  // If normal mode also fails, emit error to application
  this.emit('error', err);
}
```

## Rollback Plan

If the two-mode system causes issues:

**Immediate Rollback:**
1. Add feature flag: `ENABLE_FAST_RELOAD=false`
2. If disabled, always use normal mode
3. Keep cache but don't use it

**Partial Rollback:**
1. Keep certificate reuse (safe optimization)
2. Keep session logging
3. Disable fast path candidate injection
4. Always use normal gathering

**Complete Rollback:**
1. Revert main.go to original version
2. Revert wss-webrtc.ts to original version
3. Clear all caches on client side

**Rollback Triggers:**
- Fast mode failure rate >30%
- Connection instability >20%
- User reports connection issues
- Performance degradation

## Security Considerations

### Certificate Management
- Certificate reuse increases attack window
- Implement monthly rotation
- Log certificate changes

### Session Data
- Don't store sensitive data in sessions
- Use random session tokens
- Implement session expiration

### Cache Validation
- Validate cache structure
- Sanitize candidate data
- Check cache age before use

## Success Criteria

1. **Performance**
   - [ ] IPv6 fast reload connections complete in <300ms
   - [ ] IPv4 fast reload connections complete in <500ms
   - [ ] Normal connections complete in <2s
   - [ ] Fast mode success rate >80%
   - [ ] IPv6 connection success rate >70% (when IPv6 available)
   - [ ] Track and compare IPv6 vs IPv4 connection times

2. **Reliability**
   - [ ] Fallback mechanism works correctly
   - [ ] No connection loops
   - [ ] Cache doesn't cause stale connections

3. **User Experience**
   - [ ] Transparent switching between modes
   - [ ] Clear logging for debugging
   - [ ] No noticeable interruption on fallback

4. **Maintainability**
   - [ ] Clear separation of modes
   - [ ] Comprehensive logging
   - [ ] Easy to disable if needed

## Monitoring and Observability

### Key Metrics to Monitor

**Server Side:**
- Fast mode requests per minute
- Normal mode requests per minute
- IPv6 vs IPv4 connection counts
- Average connection time by protocol
- Average connection time by mode
- Session cache hit rate
- Certificate age (days)

**Client Side:**
- Cache hits vs misses
- Fast mode attempts by protocol (IPv6/IPv4)
- Fallback triggers by protocol
- Connection time distribution
- IPv6 availability detection success rate
- IPv6 vs IPv4 performance comparison

### Log Format

```
[FAST_RELOAD] Session=sess-123 Mode=fast Protocol=ipv6 Duration=287ms Result=success
[FAST_RELOAD] Session=sess-124 Mode=fast Protocol=ipv4 Duration=423ms Result=success
[FAST_RELOAD] Session=sess-125 Mode=fast Protocol=ipv6 Duration=2100ms Result=fallback
[NORMAL] Session=sess-126 Mode=normal Protocol=ipv4 Duration=1850ms Result=success
[CACHE] Action=load Age=45s Valid=true IPv6=2 IPv4=1 Protocol=ipv6
[CANDIDATE] IPv6 Host=2 IPv6 srflx=1 IPv4 srflx=1 IPv4 relay=0
```

## Next Steps

1. **Review and Approve Plan**
   - Stakeholder review
   - Technical feasibility assessment
   - Resource allocation

2. **Development Sprint 1** (Server Changes)
   - Implement certificate reuse
   - Add session management
   - Implement fast path
   - Unit tests

3. **Development Sprint 2** (Client Changes)
   - Implement caching system
   - Add two-mode logic
   - Implement fallback
   - Unit tests

4. **Integration and Testing**
   - Integrate both sides
   - Run integration tests
   - Performance benchmarks
   - Edge case testing

5. **Deployment**
   - Canary deployment
   - Monitor metrics
   - Gradual rollout
   - Full deployment

## Appendices

### A. Signal Message Format

**Offer with Session Token:**
```json
{
  "type": "offer",
  "sdp": "...",
  "sessionToken": "session-abc123"
}
```

**Answer with Connection Mode:**
```json
{
  "type": "answer",
  "sdp": "...",
  "connectionMode": "fast"
}
```

### B. Cache Data Structure

```typescript
interface CachedConnectionData {
  ipv6HostCandidates: RTCIceCandidateInit[];
  ipv6SrflxCandidates: RTCIceCandidateInit[];
  ipv4SrflxCandidates: RTCIceCandidateInit[];
  ipv4RelayCandidates: RTCIceCandidateInit[];
  lastSuccessfulProtocol: 'ipv6' | 'ipv4' | 'both';
  timestamp: number;  // Unix timestamp
  sessionId: string;  // Server session token
  lastConnectedIPv6?: boolean;  // Whether last connection was over IPv6
}
```

### C. Session Information Structure

```go
type SessionInfo struct {
    ClientID      string
    LastConnected time.Time
    LastIP        string
    PreferredICE  string
    SessionToken  string
    LastCandidates []webrtc.ICECandidateInit
    ConnectionCount int
}
```

## References

- WebRTC ICE Processing: https://datatracker.ietf.org/doc/html/rfc8445
- Full Cone NAT: https://en.wikipedia.org/wiki/Network_address_translation#Full_cone
- Pion WebRTC Documentation: https://pkg.go.dev/github.com/pion/webrtc/v3
- SimplePeer Documentation: https://github.com/feross/simple-peer

---

**Document Version:** 1.0  
**Last Updated:** 2024  
**Status:** Ready for Review
