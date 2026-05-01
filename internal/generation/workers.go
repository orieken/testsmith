package generation

// ClampWorkers returns a worker count that is at least 1 and at most fileCount.
// A value of zero or less is treated as 1 (sequential).
func ClampWorkers(n, fileCount int) int {
	if n < 1 {
		n = 1
	}
	if fileCount > 0 && n > fileCount {
		n = fileCount
	}
	return n
}
