package gochrome

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
)

type Command struct {
	Id     int        `json:"id"`
	Method string     `json:"method"`
	Params Parameters `json:"params"`
}

type Parameters map[string]interface{}

type Chrome struct {
	c *websocket.Conn
}

func Dial(url string) (*Chrome, error) {
	c, err := newClient(url)
	if err != nil {
		return nil, err
	}
	return &Chrome{c}, err
}

func (ch *Chrome) Send(co Command) error {
	message, err := json.Marshal(co)
	if err != nil {
		return err
	}
	err = ch.c.WriteMessage(1, message)
	return err
}

func (ch *Chrome) Close() error {
	return ch.c.Close()
}

func newClient(url string) (*websocket.Conn, error) {
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Add("Content-Type", "application/json")
	c, _, err := websocket.DefaultDialer.Dial(url, r.Header)
	if err != nil {
		return nil, err
	}
	return c, nil

}
