package device

type Device struct {
	id string
}

func (d *Device) ProcessMessage(prefix string, event string, content []byte) {
}
