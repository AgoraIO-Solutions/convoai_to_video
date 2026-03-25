package main

import (
	"bufio"
	"encoding/binary"
	"log"
	"sync"

	"go-publish-video/ipc/ipcgen"

	flatbuffers "github.com/google/flatbuffers/go"
)

var (
	childLogger  *log.Logger
	stdoutWriter *bufio.Writer
	stdoutLock   sync.Mutex
)

func sendAsyncStatusResponse(status ipcgen.ConnectionStatus, message string, details string) {
	stdoutLock.Lock()
	defer stdoutLock.Unlock()

	// First create the StatusResponsePayload
	innerBuilder := flatbuffers.NewBuilder(1024)
	msgStr := innerBuilder.CreateString(message)
	detailsStr := innerBuilder.CreateString(details)

	ipcgen.StatusResponsePayloadStart(innerBuilder)
	ipcgen.StatusResponsePayloadAddStatus(innerBuilder, status)
	ipcgen.StatusResponsePayloadAddErrorMessage(innerBuilder, msgStr)
	ipcgen.StatusResponsePayloadAddAdditionalInfo(innerBuilder, detailsStr)
	statusPayloadOffset := ipcgen.StatusResponsePayloadEnd(innerBuilder)
	innerBuilder.Finish(statusPayloadOffset)

	// Get the serialized StatusResponsePayload bytes
	statusPayloadBytes := innerBuilder.FinishedBytes()

	// Now create the outer IPCMessage with the StatusResponsePayload bytes as payload
	outerBuilder := flatbuffers.NewBuilder(len(statusPayloadBytes) + 64)

	// Create payload vector for IPCMessage
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(statusPayloadBytes))
	for i := len(statusPayloadBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(statusPayloadBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(statusPayloadBytes))

	// Create IPCMessage
	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeSTATUS_RESPONSE)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadStatus)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)

	buf := outerBuilder.FinishedBytes()
	sendFramedMessage(stdoutWriter, buf)
	if err := stdoutWriter.Flush(); err != nil {
		childLogger.Printf("ERROR flushing stdout after status response: %v", err)
	}
}

func sendAsyncErrorResponse(statusForError ipcgen.ConnectionStatus, errMsgStr string, errorDetails string) {
	sendAsyncStatusResponse(statusForError, errMsgStr, errorDetails)
}

func sendAsyncLogResponse(level ipcgen.LogLevel, messageStr string) {
	stdoutLock.Lock()
	defer stdoutLock.Unlock()

	// First create the LogResponsePayload
	innerBuilder := flatbuffers.NewBuilder(1024)
	msgStr := innerBuilder.CreateString(messageStr)

	ipcgen.LogResponsePayloadStart(innerBuilder)
	ipcgen.LogResponsePayloadAddLevel(innerBuilder, level)
	ipcgen.LogResponsePayloadAddMessage(innerBuilder, msgStr)
	logPayloadOffset := ipcgen.LogResponsePayloadEnd(innerBuilder)
	innerBuilder.Finish(logPayloadOffset)

	// Get the serialized LogResponsePayload bytes
	logPayloadBytes := innerBuilder.FinishedBytes()

	// Now create the outer IPCMessage with the LogResponsePayload bytes as payload
	outerBuilder := flatbuffers.NewBuilder(len(logPayloadBytes) + 64)

	// Create payload vector for IPCMessage
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(logPayloadBytes))
	for i := len(logPayloadBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(logPayloadBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(logPayloadBytes))

	// Create IPCMessage
	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeLOG_RESPONSE)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadLog)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)

	buf := outerBuilder.FinishedBytes()
	sendFramedMessage(stdoutWriter, buf)
	if err := stdoutWriter.Flush(); err != nil {
		childLogger.Printf("ERROR flushing stdout after log response: %v", err)
	}
}

func sendStatusResponse(status ipcgen.ConnectionStatus, errMsgStr string, addInfoStr string) {
	sendAsyncStatusResponse(status, errMsgStr, addInfoStr)
}

func sendErrorResponse(statusForError ipcgen.ConnectionStatus, errorMessage string, errorDetails string) {
	sendAsyncStatusResponse(statusForError, errorMessage, errorDetails)
}

func sendFramedMessage(writer *bufio.Writer, msg []byte) {
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(msg)))

	if _, err := writer.Write(lenBytes); err != nil {
		childLogger.Printf("Failed to write message length to writer: %v", err)
		return
	}
	if _, err := writer.Write(msg); err != nil {
		childLogger.Printf("Failed to write message payload to writer: %v", err)
	}
}
