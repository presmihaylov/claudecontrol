package vim

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"ccagent/models"
)

type VimService struct {
	state     *models.VimState
	clipboard Clipboard
}

type Clipboard interface {
	Copy(text string) error
	Paste() (string, error)
}

func NewVimService(clipboard Clipboard) *VimService {
	return &VimService{
		state:     models.NewVimState(),
		clipboard: clipboard,
	}
}

func (v *VimService) GetMode() models.VimMode {
	return v.state.Mode
}

func (v *VimService) SetMode(mode models.VimMode) {
	log.Printf("📋 Switching vim mode from %s to %s", v.state.Mode, mode)
	v.state.Mode = mode
	if mode != models.VimModeVisual {
		v.state.VisualStart = 0
		v.state.VisualEnd = 0
	}
}

func (v *VimService) ProcessCommand(input string) (string, bool, error) {
	if v.state.Mode == models.VimModeInsert {
		if input == "\x1b" {
			v.SetMode(models.VimModeNormal)
			return "", false, nil
		}
		return input, true, nil
	}

	if v.state.Mode == models.VimModeCommand {
		return v.processCommandMode(input)
	}

	return v.processNormalMode(input)
}

func (v *VimService) processNormalMode(input string) (string, bool, error) {
	switch input {
	case "i":
		v.SetMode(models.VimModeInsert)
		return "", false, nil
	case "v":
		v.SetMode(models.VimModeVisual)
		v.state.VisualStart = v.state.CursorPosition
		v.state.VisualEnd = v.state.CursorPosition
		return "", false, nil
	case ":":
		v.SetMode(models.VimModeCommand)
		v.state.CommandBuffer = ""
		return "", false, nil
	case "\x1b":
		return "", false, nil
	}

	if strings.HasPrefix(input, "y") {
		return v.processYankCommand(input)
	}

	if input == "p" {
		return v.processPasteCommand("")
	}

	if input == "P" {
		return v.processPasteCommand("before")
	}

	return "", false, nil
}

func (v *VimService) processCommandMode(input string) (string, bool, error) {
	if input == "\x1b" {
		v.SetMode(models.VimModeNormal)
		v.state.CommandBuffer = ""
		return "", false, nil
	}

	if input == "\n" || input == "\r" {
		result, shouldProcess := v.executeCommand(v.state.CommandBuffer)
		v.SetMode(models.VimModeNormal)
		v.state.CommandBuffer = ""
		return result, shouldProcess, nil
	}

	v.state.CommandBuffer += input
	return "", false, nil
}

func (v *VimService) executeCommand(cmd string) (string, bool) {
	cmd = strings.TrimSpace(cmd)
	
	if cmd == "q" || cmd == "quit" {
		return "exit", true
	}

	if cmd == "w" || cmd == "write" {
		return "save", true
	}

	if cmd == "wq" {
		return "save_and_exit", true
	}

	return "", false
}

func (v *VimService) processYankCommand(input string) (string, bool, error) {
	cmd := v.parseYankCommand(input)
	if cmd == nil {
		return "", false, nil
	}

	text := v.getTextForYank(cmd)
	if text == "" {
		return "", false, fmt.Errorf("no text to yank")
	}

	register := cmd.Register
	if register == "" {
		register = "0"
	}

	v.state.Registers[register] = text
	v.state.LastYankedText = text

	if register == "+" || register == "*" {
		if err := v.clipboard.Copy(text); err != nil {
			return "", false, fmt.Errorf("failed to copy to system clipboard: %w", err)
		}
	}

	log.Printf("📋 Yanked %d characters to register %s", len(text), register)
	return "", false, nil
}

func (v *VimService) parseYankCommand(input string) *models.VimCommand {
	yankRegex := regexp.MustCompile(`^"?([a-z0-9+*])?(\d*)y([ywbejkhlG$0^]|y)?$`)
	matches := yankRegex.FindStringSubmatch(input)
	
	if matches == nil {
		return nil
	}

	cmd := &models.VimCommand{
		Type: "yank",
	}

	if matches[1] != "" {
		cmd.Register = matches[1]
	}

	if matches[2] != "" {
		count, _ := strconv.Atoi(matches[2])
		cmd.Count = count
	}

	if matches[3] != "" {
		cmd.Motion = matches[3]
	}

	return cmd
}

func (v *VimService) getTextForYank(cmd *models.VimCommand) string {
	switch cmd.Motion {
	case "y", "":
		return "current line simulation"
	case "w":
		return "word simulation"
	case "$":
		return "to end of line simulation"
	case "0", "^":
		return "to beginning of line simulation"
	default:
		return "yanked text simulation"
	}
}

func (v *VimService) processPasteCommand(position string) (string, bool, error) {
	text := v.state.LastYankedText
	
	if text == "" {
		text = v.state.Registers["0"]
	}
	
	if text == "" {
		pastedText, err := v.clipboard.Paste()
		if err == nil && pastedText != "" {
			text = pastedText
		}
	}

	if text == "" {
		return "", false, fmt.Errorf("nothing to paste")
	}

	log.Printf("📋 Pasting %d characters", len(text))
	return text, true, nil
}

func (v *VimService) GetRegister(name string) string {
	return v.state.Registers[name]
}

func (v *VimService) SetRegister(name, value string) {
	v.state.Registers[name] = value
	if name == "0" {
		v.state.LastYankedText = value
	}
}