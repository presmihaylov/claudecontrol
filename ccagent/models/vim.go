package models

type VimMode string

const (
	VimModeNormal VimMode = "normal"
	VimModeInsert VimMode = "insert"
	VimModeVisual VimMode = "visual"
	VimModeCommand VimMode = "command"
)

type VimState struct {
	Mode           VimMode
	Registers      map[string]string
	LastYankedText string
	VisualStart    int
	VisualEnd      int
	CursorPosition int
	CommandBuffer  string
}

func NewVimState() *VimState {
	return &VimState{
		Mode:      VimModeNormal,
		Registers: make(map[string]string),
	}
}

type VimCommand struct {
	Type      string
	Count     int
	Register  string
	Motion    string
	Character string
}

type YankOperation struct {
	Text     string
	Register string
	Type     YankType
}

type YankType string

const (
	YankTypeLine      YankType = "line"
	YankTypeCharacter YankType = "character"
	YankTypeBlock     YankType = "block"
)