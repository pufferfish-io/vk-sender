package contract

type SendMessageRequest struct {
	PeerID  int64  `json:"peer_id"`
	Message string `json:"message"`
}
