package main

type MsgType byte

const (
	MsgTypeText MsgType = iota
	MsgTypeImage
	MsgTypeFile
	MsgTypeClearFile
	MsgTypeRequestFile
)
