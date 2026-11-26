package agenda

import (
	"slices"
	"testing"
	"time"
)

func TestIntervaloDesdeAbertura(t *testing.T) {
	dia := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	cfg := Configuracao{Capacidade: 1, Intervalo: 15, Horarios: []Expediente{{1, "08:05", "09:00"}}}
	todos := cfg.horarios(dia, 25, dia)
	if !slices.Equal(todos, []string{"08:05", "08:20", "08:35"}) {
		t.Fatalf("horários: %v", todos)
	}
	livres := cfg.vagas(dia, 25, []periodo{{485, 510}}, dia)
	if !slices.Equal(livres, []string{"08:35"}) {
		t.Fatalf("vagas: %v", livres)
	}
}
