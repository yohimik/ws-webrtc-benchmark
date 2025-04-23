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

const peerKey = "peer"

func main() {
	wServer := websocket.Server{}
	wServer.HandleOpen(onOpen)
	wServer.HandleData(onData)

	r := router.New()
	indexHTMLBytes, err := os.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	indexHTML := string(indexHTMLBytes)
	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/html")
		fmt.Fprintln(ctx, indexHTML)
	})
	r.GET("/ws", wServer.Upgrade)
}

func onOpen(c *websocket.Conn) {
	peer := &webrtc.PeerConnection{}
	peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		write(c, candidate.ToJSON())
	})
	channel, err := peer.CreateDataChannel("data", &webrtc.DataChannelInit{})
	if err != nil {
		return
	}
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		channel.Send(msg.Data)
	})
	c.SetUserValue(peerKey, peer)
}

type Msg struct {
	Candidate   *webrtc.ICECandidateInit
	Description *webrtc.SessionDescription
}

func onData(c *websocket.Conn, isBinary bool, data []byte) {
	if isBinary {
		c.Write(data)
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
