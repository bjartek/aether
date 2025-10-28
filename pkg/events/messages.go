package events

// FrontendPortMsg is sent when a frontend process starts listening on a port
type FrontendPortMsg struct {
	Port string
}

// Implement tea.Msg for FrontendPortMsg
func (m FrontendPortMsg) String() string {
	return "frontend_port"
}
