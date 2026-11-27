//go:build unix && !solaris && !tinygo

package profiling

import "syscall"

// profilingStderrMute points file descriptor 2 at a pipe that nothing reads.
// restore points it back and closes the descriptors the mute opened.
func profilingStderrMute() (restore func() error, err error) {
	var pipe [2]int
	if err := syscall.Pipe(pipe[:]); err != nil {
		return nil, err
	}
	syscall.CloseOnExec(pipe[0])
	syscall.CloseOnExec(pipe[1])
	saved, err := syscall.Dup(2)
	if err != nil {
		return nil, profilingDescriptorsClose(err, pipe[0], pipe[1])
	}
	syscall.CloseOnExec(saved)
	if err := profilingDescriptorReplace(pipe[1], 2); err != nil {
		return nil, profilingDescriptorsClose(err, saved, pipe[0], pipe[1])
	}
	return func() error {
		return profilingDescriptorsClose(profilingDescriptorReplace(saved, 2), saved, pipe[0], pipe[1])
	}, nil
}

// profilingDescriptorsClose closes descriptors and returns err, or when err
// is nil the first close error.
func profilingDescriptorsClose(err error, descriptors ...int) error {
	for _, descriptor := range descriptors {
		if closeErr := syscall.Close(descriptor); err == nil {
			err = closeErr
		}
	}
	return err
}
