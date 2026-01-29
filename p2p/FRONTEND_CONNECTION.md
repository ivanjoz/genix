Your frontend (browser) should:
1. Connect to the same WebSocket URL (from `credentials.json` or environment variable).
2. Send an "offer" signal to the `genix-bridge` target:
   ```json
   {
     "action": "sendSignal",
     "to": "genix-bridge",
     "data": { "type": "offer", "sdp": "..." }
   }
   ```
3. The Lambda will forward this to the current `LAPTOP_ID`.
4. The Home Lab server will respond with an "answer" via the Lambda back to the browser.
