package main

import (
	log "maunium.net/go/maulogger"
)

type MatrixListener struct {
	bridge *Bridge
	log    *log.Sublogger
	stop   chan struct{}
}

func NewMatrixListener(bridge *Bridge) *MatrixListener {
	return &MatrixListener{
		bridge: bridge,
		stop:   make(chan struct{}, 1),
		log: bridge.Log.CreateSublogger("Matrix", log.LevelDebug),
	}
}

func (ml *MatrixListener) Start() {
	for {
		select {
		case evt := <-ml.bridge.AppService.Events:
			log.Debugln("Received Matrix event:", evt)
		case <-ml.stop:
			return
		}
	}
}

func (ml *MatrixListener) Stop() {
	ml.stop <- struct{}{}
}