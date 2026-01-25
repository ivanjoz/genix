package signal

type Msg struct {
	Action string      `json:"action"`
	To     string      `json:"to,omitempty"`
	From   string      `json:"from,omitempty"`
	Data   interface{} `json:"data"`
}
