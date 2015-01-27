package gochrome

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var TabNotFound = errors.New("Tab not found")

type Command struct {
	Id     int        `json:"id"`
	Method string     `json:"method"`
	Params Parameters `json:"params"`
}

type Parameters map[string]interface{}

type Tab struct {
	Description          string `json:"description"`
	DevtoolsFrontendUrl  string `json:"devtoolsFrontendUrl"`
	FaviconUrl           string `json:"faviconUrl"`
	Id                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	Url                  string `json:"url"`
	WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

type Chrome struct {
	c              *websocket.Conn
	NetworkHandler func(Message)
	Loaded         chan bool
}

type Message struct {
	Method string `json:"method"`
	Params map[string]interface{}
}

type Result struct {
	Id     int                    `json:"id"`
	Error  map[string]interface{} `json:"error"`
	Result map[string]interface{} `json:"result"`
}

func New(url string, tab int, nh func(Message)) (*Chrome, error) {
	url, err := getTab(url, tab)
	c, err := newClient(url)
	if err != nil {
		return nil, err
	}
	loaded := make(chan bool)
	ch := &Chrome{c, nh, loaded}
	go ch.readMessages()
	return ch, err
}

func (ch *Chrome) Send(co Command) error {
	message, err := json.Marshal(co)
	if err != nil {
		return err
	}
	err = ch.c.WriteMessage(1, message)
	return err
}

func (ch *Chrome) SendSync(co Command) (Result, error) {
	message, err := json.Marshal(co)
	if err != nil {
		return Result{}, err
	}
	err = ch.c.WriteMessage(1, message)
	for {
		_, r, err := ch.c.ReadMessage()
		if err != nil {
			continue
		}

		res := Result{}
		err = json.Unmarshal(r, &res)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if res.Id == co.Id {
			return res, nil
		}

	}
}

func (ch *Chrome) readMessages() {
	for {
		_, r, err := ch.c.ReadMessage()
		if err != nil {
			continue
		}

		m := Message{}
		err = json.Unmarshal(r, &m)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if m.Method == "Page.loadEventFired" {
			ch.Loaded <- true
			close(ch.Loaded)
			break
		}

		if strings.HasPrefix(m.Method, "Network.") {
			go ch.NetworkHandler(m)
		}

	}

}

func getTab(url string, tab int) (string, error) {
	resp, err := http.Get(url + "/json")
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	t := []Tab{}
	err = json.Unmarshal(body, &t)
	if err != nil {
		return "", err
	}
	if len(t) < tab {
		return "", TabNotFound
	}

	return t[tab].WebSocketDebuggerUrl, nil
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
