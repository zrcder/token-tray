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

func confirmDialog(title, message string) bool {
	escapedMsg := strings.ReplaceAll(message, `"`, "`")
	script := fmt.Sprintf(
		`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show("%s", "TokenTray", "YesNo", "Warning") -eq "Yes"`,
		escapedMsg,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "True"
}

func infoDialog(title, message string) {
	escapedMsg := strings.ReplaceAll(message, `"`, "`")
	script := fmt.Sprintf(
		`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show("%s", "TokenTray — %s", "OK", "Information")`,
		escapedMsg, strings.ReplaceAll(title, `"`, "`"),
	)
	_ = exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}
