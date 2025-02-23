package main

import (
	"fmt"
	"os"

	"github.com/nealhardesty/j2k/internal/joystick2keyboard"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "j2k",
		Short: "Joystick to Keyboard Emulator",
		Long:  `A simple emulator that maps joystick inputs to keyboard keys.`,
		Run: func(cmd *cobra.Command, args []string) {
			emulator, err := joystick2keyboard.NewJoystick2Keyboard()
			if err != nil {
				fmt.Printf("Error initializing joystick2keyboard: %v\n", err)
				os.Exit(1)
			}

			if err := emulator.Run(); err != nil {
				fmt.Printf("Error running joystick2keyboard: %v\n", err)
				os.Exit(1)
			}
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error on Execute(): %v\n", err)
		os.Exit(1)
	}
}
