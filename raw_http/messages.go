package main

func cloneMessages(in []Message) []Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]Message, len(in))
	for i, m := range in {
		out[i] = m
		if len(m.ToolCalls) > 0 {
			out[i].ToolCalls = make([]ToolCall, len(m.ToolCalls))
			copy(out[i].ToolCalls, m.ToolCalls)
		}
	}
	return out
}
