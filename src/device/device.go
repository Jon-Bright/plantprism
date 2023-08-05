package device

type Device struct {
	id       string
	msgQueue chan *msgUnparsed
}

type msgUnparsed struct {
	prefix  string
	event   string
	content []byte
}

func (d *Device) ProcessMessage(prefix string, event string, content []byte) {
	d.msgQueue <- &msgUnparsed{prefix, event, content}
}

func (d *Device) processingLoop() {
	for {
		msg := <-d.msgQueue
		d.parseMessage(msg)
	}
}

func (d *Device) parseMessage(msg *msgUnparsed) {
}
