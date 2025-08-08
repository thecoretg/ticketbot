package ticketbot

func intToInt32Ptr(i int) *int32 {
	if i == 0 {
		return nil
	}
	val := int32(i)
	return &val
}
