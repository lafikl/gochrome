package gochrome

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/websocket"
)

var TabNotFound = errors.New("Tab not found")

type Command struct {
	Id     int        `json:"id"`
	Method string     `json:"method"`
	Params Parameters `json:"params"`
}

type Parameters map[string]interface{}

type gtab struct {
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
	c         *websocket.Conn
	listeners map[string][]chan Message
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

func New(url string, tab int) (*Chrome, error) {
	url, err := getTab(url, tab)
	c, err := newClient(url)
	if err != nil {
		return nil, err
	}
	ch := &Chrome{c, make(map[string][]chan Message, 0)}
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
		go ch.broadMsgs(m)
	}
}

// Runs in its own goroutine, to brodacst msgs
func (ch *Chrome) broadMsgs(m Message) {
	lc, ok := ch.listeners[m.Method]
	if !ok {
		return
	}

	for c := range lc {
		lc[c] <- m
	}
}

// Listen on a certain event
func (ch *Chrome) On(e string, c chan Message) (ok bool) {
	if _, ok := ch.listeners[e]; ok {
		ch.listeners[e] = append(ch.listeners[e], c)
		return true
	}
	ch.listeners[e] = append(make([]chan Message, 0), c)
	return true
}

// Remove a listener
func (ch *Chrome) Off(e string, c chan Message) {
	lc, ok := ch.listeners[e]
	if !ok {
		return
	}

	for i := range lc {
		if lc[i] == c {
			lc = append(lc[:i], lc[i+1:]...)
			break
		}
	}
}

func (ch *Chrome) Close() error {
	return ch.c.Close()
}

func getTab(url string, tab int) (string, error) {
	resp, err := http.Get(url + "/json")
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	t := []gtab{}
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
