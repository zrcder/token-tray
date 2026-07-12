//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func promptDialog(title, message string) string {
	script := fmt.Sprintf(
		`Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.Interaction]::InputBox("%s", "TokenTray — %s", "")`,
		strings.ReplaceAll(message, `"`, "`"),
		strings.ReplaceAll(title, `"`, "`"),
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return "__CANCELLED__"
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "__CANCELLED__"
	}
	return result
}
