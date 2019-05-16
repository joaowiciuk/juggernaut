package main

import "time"

type medidor struct {
	potenciaMedia float64
	intervalo     struct {
		inicio time.Time
		fim    time.Time
	}
	energia float64
}

func (m *medidor) acionar() {

}
