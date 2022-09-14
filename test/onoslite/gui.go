// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package onoslite

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"html/template"
	"net/http"
	"sync"
	"time"
)

const (
	indexPath  = "test/onoslite/index.html"
	pingPeriod = 10 * time.Second
)

var (
	upgrader = websocket.Upgrader{}
	maxID    = uint32(0)
)

var homeTemplate *template.Template

type webClient struct {
	id  uint32
	ch  chan []byte
	ctx context.Context
}

// HTTP/WS server for the web-based visualizer client
type guiServer struct {
	clients map[uint32]*webClient
	lock    sync.RWMutex
	onos    *LiteONOS
}

// NewServer creates a new HTTP/WS server for the web-based visualizer client
func newGUIServer(o *LiteONOS) *guiServer {
	return &guiServer{
		clients: make(map[uint32]*webClient),
		onos:    o,
	}
}

// Starts the HTTP/WS server
func (s *guiServer) serve() {
	http.HandleFunc("/watch", s.watchChanges)
	http.HandleFunc("/", s.home)
	if err := http.ListenAndServe(":5152", nil); err != nil {
		log.Warnf("Unable to start GUI server")
	}
}

func (s *guiServer) broadcast(data string) {
	for _, wc := range s.clients {
		wc.send(data)
	}
}

func (wc *webClient) send(data string) {
	wc.ch <- []byte(data)
}

// Sets up a new web-client relay topology events onto the web-client
func (s *guiServer) watchChanges(w http.ResponseWriter, r *http.Request) {
	log.Infof("Received new web client connection")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Unable to upgrade HTTP connection:", err)
		return
	}
	defer ws.Close()

	s.lock.Lock()
	maxID++
	wc := &webClient{
		id:  maxID,
		ch:  make(chan []byte, 1000),
		ctx: context.Background(),
	}
	s.clients[wc.id] = wc
	s.lock.Unlock()
	log.Infof("Client %d: Connected", wc.id)

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	ws.SetPongHandler(func(data string) error {
		log.Infof("Client %d: pong received", wc.id)
		return nil
	})

	s.sendExistingNodesAndEdges(wc)

WriteLoop:
	for {
		select {
		case b, ok := <-wc.ch:
			if !ok {
				log.Infof("Client %d: Event channel closed", wc.id)
				break WriteLoop
			}
			log.Infof("Sending: %s", string(b))
			err = ws.WriteMessage(websocket.TextMessage, b)
			if err != nil {
				log.Warnf("Client %d: Unable to write topo event: %v", wc.id, err)
				break WriteLoop
			}
		case <-ticker.C:
			// For now, this is merely to force failure at a later time; we expect no pongs currently
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warnf("Client %d: Unable to write ping: %v", wc.id, err)
				break WriteLoop
			}
		}
	}
	log.Infof("Client %d: Disconnected", wc.id)
}

func (s *guiServer) home(w http.ResponseWriter, r *http.Request) {
	var err error
	if homeTemplate, err = template.New("index.html").ParseFiles(indexPath); err != nil {
		log.Errorf("Unable to parse template %s: %+v", indexPath, err)
		return
	}
	_ = homeTemplate.Execute(w, "ws://"+r.Host+"/watch")
}

func (s *guiServer) sendExistingNodesAndEdges(wc *webClient) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, d := range s.onos.Devices {
		wc.send(nodeEvent("added", d.ID, "device"))
	}
	for _, l := range s.onos.Links {
		wc.send(edgeEvent("added", l.ID, stripPort(l.SrcPortID), stripPort(l.TgtPortID), "infra"))
	}
	for _, h := range s.onos.Hosts {
		wc.send(nodeEvent("added", h.MAC, "host"))
		wc.send(edgeEvent("added", h.MAC+h.Port, h.MAC, stripPort(h.Port), "edge"))
	}
}

func nodeEvent(event string, id string, kind string) string {
	return fmt.Sprintf(`{"event": "%s", "type": "node", "id": "%s", "kind": "%s"}`, event, id, kind)
}

func edgeEvent(event string, id string, src string, tgt string, kind string) string {
	return fmt.Sprintf(`{"event": "%s", "type": "edge", "id": "%s", "src": "%s", "tgt": "%s", "kind": "%s"}`,
		event, id, src, tgt, kind)
}
