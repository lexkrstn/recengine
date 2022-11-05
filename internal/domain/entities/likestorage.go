package entities

type LikeStorage interface {
	ApplyDelta() error
	GetDeltaFileSize() uint64
	ProcessActions(actions []Action) error
}
