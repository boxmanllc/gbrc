package utils

func MergeBytes(low, high uint8) uint16 {
	return uint16(high)<<8 | uint16(low)
}

func SplitBytes(val uint16) (uint8, uint8) {
	return uint8(val >> 8), uint8(val)
}
