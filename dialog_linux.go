//go:build !darwin && !windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func promptDialog(title, message string) string {
	out, err := exec.Command("zenity", "--entry",
		"--title=TokenTray",
		"--text="+message).Output()
	if err == nil {
		result := strings.TrimSpace(string(out))
		if result != "" {
			return result
		}
	}

	fmt.Printf("\n%s\n> ", message)
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return "__CANCELLED__"
	}
	return input
}

func confirmDialog(title, message string) bool {
	err := exec.Command("zenity", "--question",
		"--title=TokenTray",
		"--text="+message).Run()
	return err == nil
}
