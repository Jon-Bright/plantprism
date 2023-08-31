package device

// This is outgoing only, no incoming equivalent

// Example: {"cmd": "mcu_trigger_water_event", "layer": "layer_b"}
type msgAglRPCPut struct {
	Cmd   string `json:"cmd"`
	Layer string `json:"layer"`
}

func (m *msgAglRPCPut) topic() string {
	return MQTT_TOPIC_AGL_RPC_PUT
}

func getAglRPCPutWatering(l layerID) msgReply {
	msg := msgAglRPCPut{
		Cmd:   "mcu_trigger_water_event",
		Layer: "layer_" + string(l),
	}
	return &msg

}
