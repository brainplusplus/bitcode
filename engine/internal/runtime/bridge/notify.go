package bridge

import (
	"strings"

	"github.com/bitcode-framework/bitcode/internal/presentation/websocket"
)

type notifyBridge struct {
	hub     *websocket.Hub
	session Session
}

func newNotifyBridge(hub *websocket.Hub, session Session) *notifyBridge {
	return &notifyBridge{hub: hub, session: session}
}

func (n *notifyBridge) Send(opts NotifyOptions) error {
	data := map[string]any{
		"title":   opts.Title,
		"message": opts.Message,
		"type":    opts.Type,
	}

	target := opts.To
	if strings.HasPrefix(target, "user:") {
		userID := strings.TrimPrefix(target, "user:")
		n.hub.Broadcast("notification:"+userID, data)
	} else {
		n.hub.Broadcast("notification:"+target, data)
	}
	return nil
}

func (n *notifyBridge) Broadcast(channel string, data map[string]any) error {
	if n.session.TenantID != "" {
		n.hub.BroadcastToTenant(n.session.TenantID, channel, data)
	} else {
		n.hub.Broadcast(channel, data)
	}
	return nil
}
