package utils

func MergeBytes(low, high uint8) uint16 {
	return uint16(high)<<8 | uint16(low)
}
