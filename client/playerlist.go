package client

import (
	"github.com/Tnze/go-mc/server"
)

// Implement server.PlayerListClient interface
func (c *Client) ClientJoin(sample server.PlayerSample) {
	// Implementation
}

func (c *Client) ClientLeft() {
	// Implementation
}

func (c *Client) ClientTick() {
	// Implementation
}

// Implement server.KeepAliveClient interface
func (c *Client) AddPlayer() {
	// Implementation
}

func (c *Client) RemovePlayer() {
	// Implementation
}
