package fastqueue

func fnv1a64(b []byte) uint64 {
	var h uint64 = 1469598103934665603
	const p uint64 = 1099511628211
	for _, c := range b { h ^= uint64(c); h *= p }
	return h
}

func ShardFor(key []byte, n int) int {
	if n <= 1 { return 0 }
	return int(fnv1a64(key) & uint64(n-1))
}
