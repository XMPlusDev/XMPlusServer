package controller

type ControllerInterface interface {
	Start() error
	Close() error
}

type TriggerInterface interface {
	TriggerNodeSync()
	TriggerSubscriptionSync()
	GetNodeID() int
}
