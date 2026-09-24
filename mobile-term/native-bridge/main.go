package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"github.com/gorilla/websocket"
	"golang.org/x/sys/windows"
)

type InputEvent struct {
	Type     string  `json:"type"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	DeltaX   float64 `json:"deltaX,omitempty"`
	DeltaY   float64 `json:"deltaY,omitempty"`
	Button   int     `json:"button,omitempty"`
	Buttons  int     `json:"buttons,omitempty"`
}

type WsMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1
	MOUSEEVENTF_MOVE        = 0x0001
	MOUSEEVENTF_LEFTDOWN    = 0x0002
	MOUSEEVENTF_LEFTUP      = 0x0004
	MOUSEEVENTF_RIGHTDOWN   = 0x0008
	MOUSEEVENTF_RIGHTUP     = 0x0010
	MOUSEEVENTF_WHEEL       = 0x0800
	KEYEVENTF_KEYUP         = 0x0002
)

type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uint64
}

type KEYBDINPUT struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	DwExtraInfo uint64
}

type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
	Ki   KEYBDINPUT
}

var (
	moduser32     = windows.NewLazySystemDLL("user32.dll")
	procSendInput = moduser32.NewProc("SendInput")
)

var keyMap = map[string]uint16{
	"ctrl":    0x11,
	"alt":    0x12,
	"win":    0x5B,
	"tab":    0x09,
	"enter":  0x0D,
	"escape": 0x1B,
	"f5":     0x74,
	"c":      0x43,
	"v":      0x56,
	"x":      0x58,
	"z":      0x5A,
	"a":      0x41,
}

func main() {
	if runtime.GOOS != "windows" {
		log.Fatal("native-bridge only runs on Windows")
	}

	signalingURL := os.Getenv("SIGNALING_URL")
	if signalingURL == "" {
		signalingURL = "ws://localhost:3000/bridge"
	}

	log.Printf("Signaling URL: %s", signalingURL)

	c, _, err := websocket.DefaultDialer.Dial(signalingURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	go func() {
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				log.Println("ws read error:", err)
				return
			}
			var msg WsMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				log.Println("bad ws message:", err)
				continue
			}
			if msg.Type == "input" && msg.Payload != "" {
				var ev InputEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err == nil {
					injectInput(ev)
				}
			} else if msg.Type == "keyboard" && msg.Payload != "" {
				var kb map[string]string
				if err := json.Unmarshal([]byte(msg.Payload), &kb); err == nil {
					injectKeyboard(kb["keys"])
				}
			}
		}
	}()

	select {}
}

func injectInput(ev InputEvent) {
	var input INPUT

	switch ev.Type {
	case "mousedown":
		input.Type = INPUT_MOUSE
		input.Mi.Dx = int32(ev.X)
		input.Mi.Dy = int32(ev.Y)
		input.Mi.MouseData = 0
		input.Mi.DwFlags = MOUSEEVENTF_LEFTDOWN | MOUSEEVENTF_MOVE
		input.Mi.Time = 0
		input.Mi.DwExtraInfo = 0
		callSendInput(&input)
	case "mouseup":
		input.Type = INPUT_MOUSE
		input.Mi.Dx = int32(ev.X)
		input.Mi.Dy = int32(ev.Y)
		input.Mi.MouseData = 0
		input.Mi.DwFlags = MOUSEEVENTF_LEFTUP | MOUSEEVENTF_MOVE
		input.Mi.Time = 0
		input.Mi.DwExtraInfo = 0
		callSendInput(&input)
	case "mousemove":
		input.Type = INPUT_MOUSE
		input.Mi.Dx = int32(ev.X)
		input.Mi.Dy = int32(ev.Y)
		input.Mi.MouseData = 0
		input.Mi.DwFlags = MOUSEEVENTF_MOVE
		input.Mi.Time = 0
		input.Mi.DwExtraInfo = 0
		callSendInput(&input)
	case "wheel":
		input.Type = INPUT_MOUSE
		input.Mi.Dx = 0
		input.Mi.Dy = 0
		input.Mi.MouseData = uint32(ev.DeltaY)
		input.Mi.DwFlags = MOUSEEVENTF_WHEEL
		input.Mi.Time = 0
		input.Mi.DwExtraInfo = 0
		callSendInput(&input)
	}
}

func injectKeyboard(keysStr string) {
	parts := strings.Split(strings.ToLower(keysStr), "+")
	if len(parts) == 0 {
		return
	}

	var inputs []INPUT
	for _, p := range parts {
		p = strings.TrimSpace(p)
		vk, ok := keyMap[p]
		if !ok {
			continue
		}
		inputs = append(inputs, INPUT{
			Type: INPUT_KEYBOARD,
			Ki: KEYBDINPUT{
				Vk:        vk,
				Scan:      0,
				Flags:     0,
				Time:      0,
				DwExtraInfo: 0,
			},
		})
	}

	for i := range inputs {
		callSendInput(&inputs[i])
	}

	for i := len(inputs) - 1; i >= 0; i-- {
		inputs[i].Ki.Flags = KEYEVENTF_KEYUP
		callSendInput(&inputs[i])
	}
}

func callSendInput(input *INPUT) {
	procSendInput.Call(1, uintptr(unsafe.Pointer(input)), uintptr(unsafe.Sizeof(*input)))
}
