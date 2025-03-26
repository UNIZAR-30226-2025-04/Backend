package handlers

import (
	"Nogler/services/poker"
)

// Hacer aquí una tablica de relación nombre -> puntos o devolver desde el otro lado
// Un valor directamente. Lo mejor sería que consultemos en un spot (redis pg o dnd  sea)
// El nivel al que tenemo sla mano para saber fichas y mult base
// Ahora mismo está como string en el aproach mencionado sería 2 ints, fichas y mult
func PlayHand(h poker.Hand) (int, int) {
	return poker.BestHand(h)
}

func ApplyJokers(h poker.Hand, fichas int, mult int) int {
	// Given a hand and the points obtained from poker.Hand
	return 1
}
