package Error

import "errors"

var (
	ErrPoolAlreadyRegistered = errors.New("pool name already registered")
	ErrPoolNotFound          = errors.New("pool not found")
)

var (
	ErrTimerAlreadyRunning    = errors.New("timer already running")
	ErrNoKeyFrames            = errors.New("no key frames")
	ErrInvalidTimerParameters = errors.New("invalid parameters")
	ErrActorNotSet            = errors.New("actor not set")
)

var (
	ErrKeyFramePoolNotRegistered = errors.New("keyframe pool not registered")
	ErrInvalidKeyFrameTime       = errors.New("invalid keyframe time (must be > 0)")
	ErrNilKeyFrameAction         = errors.New("keyframe action cannot be nil")
	ErrKeyFrameDoubleRelease     = errors.New("attempted to release keyframe twice")
)
