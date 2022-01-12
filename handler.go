package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nhooyr.io/websocket"
)

var (
	HTTPHandler = http.NewServeMux()

	idMatcher, _ = regexp.Compile(`/(\d+)$`)
)

func init() {
	HTTPHandler.Handle("/", http.HandlerFunc(index))
	HTTPHandler.Handle("/id", http.HandlerFunc(id))
	HTTPHandler.Handle("/upload/", http.HandlerFunc(upload))
	HTTPHandler.Handle("/download/", http.HandlerFunc(download))
	HTTPHandler.Handle("/ws", http.HandlerFunc(ws))
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/index.") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; img-src blob:; script-src 'unsafe-inline'; style-src 'unsafe-inline'")
		w.Write([]byte(WebPageTemplate))
	}
}

func id(w http.ResponseWriter, _ *http.Request) {
	ID := getFileId()
	json.NewEncoder(w).Encode(struct {
		ID uint32 `json:"id"`
	}{ID})
}

func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only support POST", http.StatusMethodNotAllowed)
		return
	}
	match := idMatcher.FindStringSubmatch(r.URL.Path)
	if len(match) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(match[1], 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	uploadFile(uint32(id), w, r)
}

func download(w http.ResponseWriter, r *http.Request) {
	match := idMatcher.FindStringSubmatch(r.URL.Path)
	if len(match) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(match[1], 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	requestFile(uint32(id), w, r)
}

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "unhandled server error")
	c.SetReadLimit(int64(*messageSizeLimit))

	name := r.URL.Query().Get("name")
	if name == "" {
		name = r.RemoteAddr
	}
	nameLen := byte(len(name))
	byteName := []byte(name[:nameLen])

	addSubscriber(c)
	defer delSubscriber(c)
	defer clearFile(c)

	ctx, close := context.WithCancel(r.Context())

	his := history.Front()
	for his != nil {
		c.Write(ctx, websocket.MessageBinary, his.Value.([]byte))
		his = his.Next()
	}

	go func() {
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				if !(websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway) {
					log.Println(err)
				}
				close()
				return
			}

			msg := make([]byte, 1+int(nameLen)+8+len(data))

			offset := 0
			msg[offset] = data[0]
			offset++
			msg[offset] = nameLen
			offset++
			copy(msg[offset:], byteName)
			offset += int(nameLen)
			now := dateNow()
			copy(msg[offset:], now[:])
			offset += len(now)

			mt := MsgType(data[0])

			switch mt {
			case MsgTypeFile:
				idByte := data[1:5]
				var id uint32 = 0
				for i := 3; i >= 0; i-- {
					id += uint32(idByte[i]) * uint32(math.Pow(2, float64((3-i)*8)))
				}
				copy(msg[offset:], data[1:])

				msgObj := publish(ctx, msg, true)
				newFile(c, id, msgObj)
			default:
				copy(msg[offset:], data[1:])
				publish(ctx, msg, true)
			}
		}
	}()

	// ticker := time.NewTicker(10 * time.Second)
	// defer ticker.Stop()
	// go func() {
	// 	for {
	// 		<-ticker.C
	// 		if err := c.Ping(ctx); err != nil {
	// 			log.Println(err.Error())
	// 			close()
	// 		}
	// 	}
	// }()

	<-ctx.Done()
}

func dateNow() (now [8]byte) {
	t := time.Now().UnixMilli()
	for i := 7; i >= 0; i-- {
		now[i] = byte(t & 0xFF)
		t >>= 8
	}
	return
}

func uint32ToBytes(num uint32) (result [4]byte) {
	for i := 3; i >= 0; i-- {
		result[i] = byte(num & 0xFF)
		num >>= 8
	}
	return
}
