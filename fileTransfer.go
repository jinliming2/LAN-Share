package main

import (
	"bufio"
	"container/list"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type fileReceiver struct {
	w          http.ResponseWriter
	r          *http.Request
	cancelWait func()
	done       chan bool
}

var (
	fileCounter   uint32 = 0
	fileCounterMu        = sync.Mutex{}

	file2Subscriber  = make(map[*websocket.Conn]map[uint32]*list.Element)
	id2File          = make(map[uint32]*websocket.Conn)
	fileSubscriberMu = sync.RWMutex{}

	pendingTransfer   = make(map[uint32]*list.List)
	pendingTransferMu = sync.RWMutex{}
)

func getFileId() (id uint32) {
	fileCounterMu.Lock()
	defer fileCounterMu.Unlock()
	id = fileCounter
	fileCounter++
	return
}

func newFile(subscriber *websocket.Conn, id uint32, msgObj *list.Element) {
	fileSubscriberMu.Lock()
	defer fileSubscriberMu.Unlock()

	if _, ok := file2Subscriber[subscriber]; !ok {
		file2Subscriber[subscriber] = make(map[uint32]*list.Element)
	}
	file2Subscriber[subscriber][id] = msgObj
	id2File[id] = subscriber
}

func clearFile(subscriber *websocket.Conn) {
	fileSubscriberMu.Lock()
	defer fileSubscriberMu.Unlock()
	pendingTransferMu.Lock()
	defer pendingTransferMu.Unlock()

	clearFileMsg := []byte{byte(MsgTypeClearFile)}

	if subList, ok := file2Subscriber[subscriber]; ok {
		for id, msgObj := range subList {
			delete(id2File, id)
			history.Remove(msgObj)
			idByte := uint32ToBytes(id)
			clearFileMsg = append(clearFileMsg, idByte[:]...)
			if l, ok := pendingTransfer[id]; ok {
				for item := l.Front(); item != nil; item = item.Next() {
					r := item.Value.(*fileReceiver)
					r.cancelWait()
					http.NotFound(r.w, r.r)
					r.done <- true
				}
				delete(pendingTransfer, id)
			}
		}
	}

	publish(context.Background(), clearFileMsg, false)
}

func requestFile(id uint32, w http.ResponseWriter, r *http.Request) {
	fileSubscriberMu.RLock()
	subscriber, ok := id2File[id]
	fileSubscriberMu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	ctx, cancelWait := context.WithTimeout(r.Context(), 5*time.Second)

	pendingTransferMu.Lock()
	if _, ok := pendingTransfer[id]; !ok {
		pendingTransfer[id] = list.New()
	}
	done := make(chan bool, 1)
	item := &fileReceiver{
		w,
		r,
		cancelWait,
		done,
	}
	elem := pendingTransfer[id].PushBack(item)
	pendingTransferMu.Unlock()

	requestRange := r.Header.Get("Range")
	msg := make([]byte, 1+4+len(requestRange))
	idByte := uint32ToBytes(id)
	msg[0] = byte(MsgTypeRequestFile)
	copy(msg[1:], idByte[:])
	if len(requestRange) > 0 {
		copy(msg[5:], []byte(requestRange))
	}
	subscriber.Write(ctx, websocket.MessageBinary, msg)

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		http.Error(w, "Request Timeout", http.StatusRequestTimeout)
	} else {
		<-done
	}

	pendingTransferMu.Lock()
	defer pendingTransferMu.Unlock()
	pendingTransfer[id].Remove(elem)
}

func uploadFile(id uint32, w http.ResponseWriter, r *http.Request) {
	pendingTransferMu.RLock()

	l, ok := pendingTransfer[id]
	if !ok || l.Len() == 0 {
		pendingTransferMu.RUnlock()
		http.NotFound(w, r)
		return
	}

	requestRange := r.URL.Query().Get("range")
	name := strings.ReplaceAll(r.URL.Query().Get("name"), `"`, `\"`)
	size := r.URL.Query().Get("size")
	contentType := r.URL.Query().Get("type")
	contentRange := r.Header.Get("Content-Range")

	if name == "" {
		name = strconv.Itoa(int(id))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	receiver := make([]io.Writer, 0, l.Len())
	done := make([]chan bool, 0, l.Len())
	for item := l.Front(); item != nil; item = item.Next() {
		r := item.Value.(*fileReceiver)
		if r.r.Header.Get("Range") != requestRange {
			continue
		}
		r.cancelWait()
		receiver = append(receiver, r.w.(io.Writer))
		done = append(done, r.done)
		r.w.Header().Set("Content-Type", contentType)
		method := "attachment"
		if r.r.URL.Query().Has("open") {
			method = "inline"
		}
		r.w.Header().Set("Content-Disposition", method+`; filename="`+name+`"`)
		if size != "" {
			r.w.Header().Set("Content-Length", size)
		}
		if contentRange != "" {
			r.w.Header().Set("Content-Range", contentRange)
		}
		r.w.WriteHeader(http.StatusPartialContent)
	}

	pendingTransferMu.RUnlock()

	defer func() {
		for _, d := range done {
			d <- true
		}
	}()

	writer := newMultiWriterIgnoreError(receiver...)
	if _, err := bufio.NewReader(r.Body).WriteTo(writer); err != nil {
		http.Error(w, err.Error(), http.StatusAccepted)
		return
	}
}

type multiWriterIgnoreError struct {
	writers []io.Writer
	errors  map[int]interface{}
}

func newMultiWriterIgnoreError(writers ...io.Writer) io.Writer {
	return &multiWriterIgnoreError{
		writers: writers,
		errors:  make(map[int]interface{}),
	}
}

func (mwie *multiWriterIgnoreError) Write(p []byte) (n int, err error) {
	for i, w := range mwie.writers {
		if _, ok := mwie.errors[i]; ok {
			continue
		}

		n, err = w.Write(p)
		if err != nil {
			mwie.errors[i] = nil
			continue
		}
		if n != len(p) {
			mwie.errors[i] = nil
			continue
		}
	}
	if len(mwie.errors) == len(mwie.writers) {
		return 0, errors.New("failed to write")
	}
	return len(p), nil
}
