package cache

import (
	"bytes"
	"sync"
)

type Cache struct {
	infoMU      sync.RWMutex
	playersMU   sync.RWMutex
	infoResp    []byte
	playersResp []byte
}

func New() *Cache {
	return &Cache{}
}

func (c *Cache) SetInfoResponse(resp []byte) {
	c.infoMU.Lock()
	defer c.infoMU.Unlock()

	c.infoResp = resp
}

func (c *Cache) InfoResponse() []byte {
	c.infoMU.RLock()
	defer c.infoMU.RUnlock()

	return bytes.Clone(c.infoResp)
}

func (c *Cache) SetPlayersResponse(resp []byte) {
	c.playersMU.Lock()
	defer c.playersMU.Unlock()

	c.playersResp = resp
}

func (c *Cache) PlayersResponse() []byte {
	c.playersMU.RLock()
	defer c.playersMU.RUnlock()

	return bytes.Clone(c.playersResp)
}
