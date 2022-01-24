package sphero

import "time"

// DoFor executes the given function and waits for the duration
func DoFor(d time.Duration, f func()) {
	f()
	time.Sleep(d)
}

// DoWithDelay executes the given steps and waits for the given duration between steps
func DoWithDelay(d time.Duration, steps ...func()) {
	for _, f := range steps {
		f()
		time.Sleep(d)
	}
}
