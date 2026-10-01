//go:build darwin

package eventkit

import "C"

// changes receives raw EKEventStoreChangedNotification signals.
var changes = make(chan struct{}, 1)

//export sundialStoreChanged
func sundialStoreChanged() {
	select {
	case changes <- struct{}{}:
	default:
	}
}
