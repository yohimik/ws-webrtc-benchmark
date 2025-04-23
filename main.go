package main

import (
	"encoding/json"
	"fmt"
	"github.com/dgrr/websocket"
	"github.com/fasthttp/router"
	"github.com/pion/webrtc/v4"
	"github.com/valyala/fasthttp"
	"io"
	"os"
)

type Msg struct {
	Candidate   *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Description *webrtc.SessionDescription `json:"description,omitempty"`
}

const peerKey = "peer"

func main() {
	wServer := websocket.Server{}
	wServer.HandleOpen(onOpen)
	wServer.HandleData(onData)
	r := router.New()
	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		indexHTMLBytes, err := os.ReadFile("index.html")
		if err != nil {
			panic(err)
		}
		indexHTML := string(indexHTMLBytes)
		ctx.SetContentType("text/html")
		fmt.Fprintln(ctx, indexHTML)
	})
	r.GET("/ws", wServer.Upgrade)
	server := fasthttp.Server{
		Handler: r.Handler,
	}
	server.ListenAndServe(":8080")
}

func onOpen(c *websocket.Conn) {
	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return
	}
	peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		candidateJSON := candidate.ToJSON()
		write(c, &Msg{Candidate: &candidateJSON})
	})
	channel, err := peer.CreateDataChannel("data", &webrtc.DataChannelInit{})
	if err != nil {
		return
	}
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		channel.Send(msg.Data)
	})
	offer, err := peer.CreateOffer(nil)
	if err != nil {
		return
	}
	err = peer.SetLocalDescription(offer)
	if err != nil {
		return
	}
	c.SetUserValue(peerKey, peer)
	write(c, &Msg{Description: &offer})
}

func onData(c *websocket.Conn, isBinary bool, data []byte) {
	if isBinary {
		fr := websocket.AcquireFrame()
		fr.SetFin()
		fr.SetPayload(data)
		fr.SetBinary()
		c.WriteFrame(fr)
		return
	}
	msg := &Msg{}
	err := json.Unmarshal(data, msg)
	if err != nil {
		return
	}
	conn := c.UserValue(peerKey).(*webrtc.PeerConnection)
	if msg.Description != nil {
		conn.SetRemoteDescription(*msg.Description)
	}
	if msg.Candidate != nil {
		conn.AddICECandidate(*msg.Candidate)
	}
}

func write(w io.Writer, data any) {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return
	}
	w.Write(marshalled)
}
