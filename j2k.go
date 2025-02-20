package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/karalabe/hid"
	"github.com/micmonay/keybd_event"
)

type Joystick2Keyboard struct {
	running      bool
	deadzone     float64
	keyMappings  map[string]int
	buttonStates map[int]bool
	kb           keybd_event.KeyBonding
	// lastLX       float64
	// lastLY       float64
	// lastRX       float64
	// lastRY       float64
	mu sync.Mutex
}

// Controller state data format
type ControllerState struct {
	buttons uint16 // First 16 buttons
	leftX   int16  // Left stick X
	leftY   int16  // Left stick Y
	rightX  int16  // Right stick X
	rightY  int16  // Right stick Y
	trigL   uint8  // Left trigger
	trigR   uint8  // Right trigger
}

func NewJoystick2Keyboard() (*Joystick2Keyboard, error) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keyboard: %v", err)
	}

	// Initialize with default mappings
	keyMappings := map[string]int{
		// Left stick
		"lstick_left":  keybd_event.VK_A,
		"lstick_right": keybd_event.VK_D,
		"lstick_up":    keybd_event.VK_W,
		"lstick_down":  keybd_event.VK_S,

		// Right stick
		"rstick_left":  keybd_event.VK_J,
		"rstick_right": keybd_event.VK_L,
		"rstick_up":    keybd_event.VK_I,
		"rsdown":       keybd_event.VK_K,

		// D-pad
		"dpad_up":    keybd_event.VK_UP,
		"dpad_down":  keybd_event.VK_DOWN,
		"dpad_left":  keybd_event.VK_LEFT,
		"dpad_right": keybd_event.VK_RIGHT,

		// Face buttons
		"a": keybd_event.VK_SPACE,
		"b": keybd_event.VK_E,
		"y": keybd_event.VK_Q,
		"x": keybd_event.VK_F,

		// Shoulder buttons and triggers
		"rbutton": keybd_event.VK_R,
		"lbutton": keybd_event.VK_T,

		// Triggers
		"rtrigger": keybd_event.VK_Y,
		"ltrigger": keybd_event.VK_U,

		// Select/Start buttons
		"select": keybd_event.VK_TAB,
		"start":  keybd_event.VK_ENTER,

		// L3/R3 buttons
		"lstick_click": keybd_event.VK_L,
		"rstick_click": keybd_event.VK_R,
	}

	buttonStates := make(map[int]bool)
	for _, key := range keyMappings {
		buttonStates[key] = false
	}

	return &Joystick2Keyboard{
		running:      true,
		deadzone:     0.2,
		keyMappings:  keyMappings,
		buttonStates: buttonStates,
		kb:           kb,
		mu:           sync.Mutex{},
	}, nil
}

func (j *Joystick2Keyboard) normalizeAxis(value int16) float64 {
	return float64(value) / 32768.0
}

func (j *Joystick2Keyboard) processLeftStick(x, y float64) {
	// Apply deadzone
	magnitude := math.Sqrt(x*x + y*y)
	if magnitude < j.deadzone {
		x, y = 0, 0
	}

	// Normalize values
	if magnitude > 0 {
		x = x / magnitude
		y = y / magnitude
	}

	// Update keyboard states based on stick position
	j.updateKeyState("lstick_left", x < -0.5)
	j.updateKeyState("lstick_right", x > 0.5)
	j.updateKeyState("lstick_up", y < -0.5)
	j.updateKeyState("lstick_down", y > 0.5)
}

func (j *Joystick2Keyboard) processRightStick(x, y float64) {
	// Apply deadzone
	magnitude := math.Sqrt(x*x + y*y)
	if magnitude < j.deadzone {
		x, y = 0, 0
	}

	// Normalize values
	if magnitude > 0 {
		x = x / magnitude
		y = y / magnitude
	}

	// Update keyboard states based on stick position
	j.updateKeyState("rstick_left", x < -0.5)
	j.updateKeyState("rstick_right", x > 0.5)
	j.updateKeyState("rstick_up", y < -0.5)
	j.updateKeyState("rstick_down", y > 0.5)
}

func (j *Joystick2Keyboard) updateKeyState(buttonName string, pressed bool) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if key, exists := j.keyMappings[buttonName]; exists {
		if pressed && !j.buttonStates[key] {
			fmt.Printf("Pressing key: %s\n", buttonName)
			j.kb.SetKeys(key)
			j.kb.Press()
			j.buttonStates[key] = true
		} else if !pressed && j.buttonStates[key] {
			j.kb.SetKeys(key)
			j.kb.Release()
			j.buttonStates[key] = false
		}
	}
}

func (j *Joystick2Keyboard) processButtons(buttons uint16) {
	// Process D-pad (first 4 bits)
	j.updateKeyState("dpad_up", (buttons&0x0001) != 0)
	j.updateKeyState("dpad_down", (buttons&0x0002) != 0)
	j.updateKeyState("dpad_left", (buttons&0x0004) != 0)
	j.updateKeyState("dpad_right", (buttons&0x0008) != 0)

	// Process face buttons
	j.updateKeyState("a", (buttons&0x0010) != 0)
	j.updateKeyState("b", (buttons&0x0020) != 0)
	j.updateKeyState("x", (buttons&0x0040) != 0)
	j.updateKeyState("y", (buttons&0x0080) != 0)

	// Process shoulder buttons
	j.updateKeyState("lb", (buttons&0x0100) != 0)
	j.updateKeyState("rb", (buttons&0x0200) != 0)

	// Process select/start buttons
	j.updateKeyState("select", (buttons&0x0400) != 0)
	j.updateKeyState("start", (buttons&0x0800) != 0)

	// Process L3/R3 buttons (pressing down the analog sticks)
	j.updateKeyState("l3", (buttons&0x1000) != 0) // Left stick
	j.updateKeyState("r3", (buttons&0x2000) != 0) // Right stick

}

func (j *Joystick2Keyboard) Run() error {
	// Initialize HID

	// Find Xbox 360 controller
	var device *hid.Device
	devices := hid.Enumerate(0x045E, 0x028E) // Microsoft Xbox 360 controller VID/PID
	if len(devices) == 0 {
		return fmt.Errorf("no compatible controller found")
	}

	// Open the first found device
	var err error
	device, err = devices[0].Open()
	if err != nil {
		return fmt.Errorf("failed to open controller: %v", err)
	}
	defer device.Close()

	fmt.Println("Controller connected. Starting keyboard emulation...")
	fmt.Println("\nCurrent mappings:")
	fmt.Println("\nAnalog Sticks:")
	fmt.Println("Left Stick  -> WASD")
	fmt.Println("Right Stick -> IJKL")
	fmt.Println("\nD-pad:")
	fmt.Println("Up    -> Up Arrow")
	fmt.Println("Down  -> Down Arrow")
	fmt.Println("Left  -> Left Arrow")
	fmt.Println("Right -> Right Arrow")
	fmt.Println("\nButtons:")
	fmt.Println("A -> Space")
	fmt.Println("B -> E")
	fmt.Println("X -> F")
	fmt.Println("Y -> Q")
	fmt.Println("\nTriggers/Shoulders:")
	fmt.Println("LB -> Shift")
	fmt.Println("RB -> R")
	fmt.Println("LT -> Alt")
	fmt.Println("RT -> Ctrl")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		fmt.Println("\nStopping emulator...")
		j.Stop()
		os.Exit(0)
	}()

	// Main input loop
	buffer := make([]byte, 64)
	for j.running {
		n, err := device.Read(buffer)
		if err != nil {
			fmt.Printf("Error reading from controller: %v\n", err)
			continue
		}

		if n < 14 { // Minimum size for controller state
			continue
		}

		state := ControllerState{}
		state.buttons = binary.LittleEndian.Uint16(buffer[0:2])
		state.leftX = int16(binary.LittleEndian.Uint16(buffer[2:4]))
		state.leftY = int16(binary.LittleEndian.Uint16(buffer[4:6]))
		state.rightX = int16(binary.LittleEndian.Uint16(buffer[6:8]))
		state.rightY = int16(binary.LittleEndian.Uint16(buffer[8:10]))
		state.trigL = buffer[10]
		state.trigR = buffer[11]

		// Process all inputs
		j.processButtons(state.buttons)
		j.processLeftStick(j.normalizeAxis(state.leftX), j.normalizeAxis(state.leftY))
		j.processRightStick(j.normalizeAxis(state.rightX), j.normalizeAxis(state.rightY))
		j.updateKeyState("lt", float64(state.trigL)/255.0 > 0.5)
		j.updateKeyState("rt", float64(state.trigR)/255.0 > 0.5)

		time.Sleep(time.Millisecond * 16) // ~60Hz polling rate
	}

	return nil
}

func (j *Joystick2Keyboard) Stop() {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.running = false
	// Release all pressed keys
	for key, pressed := range j.buttonStates {
		if pressed {
			j.kb.SetKeys(key)
			j.kb.Release()
			j.buttonStates[key] = false
		}
	}
}

func main() {
	emulator, err := NewJoystick2Keyboard()
	if err != nil {
		fmt.Printf("Error initializing emulator: %v\n", err)
		os.Exit(1)
	}

	if err := emulator.Run(); err != nil {
		fmt.Printf("Error running emulator: %v\n", err)
		os.Exit(1)
	}
}
