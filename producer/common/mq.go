package common

type MQ interface {
	Input() chan<- []byte
	Close()
	SetFailMode()
}
