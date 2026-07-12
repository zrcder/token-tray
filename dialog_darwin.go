//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func promptDialog(title, message string) string {
	script := fmt.Sprintf(`
set dialogResult to display dialog "%s" default answer "" with title "TokenTray — %s" buttons {"取消", "保存"} default button "保存"
if button returned of dialogResult = "保存" then
	return text returned of dialogResult
end if
return "__CANCELLED__"
`, escapeDialog(message), escapeDialog(title))

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "__CANCELLED__"
	}
	return strings.TrimSpace(strings.TrimRight(string(out), "\n"))
}

func escapeDialog(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
