package agenda

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// DiaSemana segue o calendário ISO: segunda = 1, domingo = 7.
type Expediente struct {
	DiaSemana int    `json:"diaSemana"`
	Abre      string `json:"abre"`
	Fecha     string `json:"fecha"`
}

type Configuracao struct {
	Tolerancia int          `json:"toleranciaMinutos"`
	Capacidade int          `json:"capacidade"`
	Intervalo  int          `json:"intervaloMinutos"`
	Horarios   []Expediente `json:"horarios"`
}

type periodo struct{ inicio, fim int }

func minuto(hora string) (int, bool) {
	if hora == "24:00" {
		return 1440, true
	}
	t, err := time.Parse("15:04", hora)
	return t.Hour()*60 + t.Minute(), err == nil && len(hora) == 5
}

func relogio(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

func (c Configuracao) validar() error {
	if c.Tolerancia < 0 || c.Tolerancia > 1440 || c.Capacidade < 1 || c.Capacidade > 2147483647 || c.Intervalo < 1 || c.Intervalo > 1440 || c.Horarios == nil {
		return errors.New("Informe capacidade, intervalo em minutos e horários da semana")
	}
	var aberto [7][1440]bool
	for _, h := range c.Horarios {
		ini, okInicio := minuto(h.Abre)
		fim, okFim := minuto(h.Fecha)
		if !okInicio || !okFim || ini >= fim || h.DiaSemana < 1 || h.DiaSemana > 7 {
			return errors.New("Informe um dia de 1 a 7 e um período válido em HH:MM")
		}
		for m := ini; m < fim; m++ {
			if aberto[h.DiaSemana-1][m] {
				return errors.New("Há períodos de funcionamento sobrepostos")
			}
			aberto[h.DiaSemana-1][m] = true
		}
	}
	return nil
}

// Calcula a ocupação minuto a minuto. O fim de uma reserva já está livre.
func (c Configuracao) vagas(dia time.Time, duracao int, reservas []periodo, agora time.Time) []string {
	vagas := []string{}
	if c.Intervalo <= 0 || duracao <= 0 || c.Capacidade <= 0 || dia.AddDate(0, 0, 1).Before(agora) {
		return vagas
	}
	var variacao [1441]int
	for _, p := range reservas {
		variacao[p.inicio]++
		variacao[p.fim]--
	}
	var lotados [1441]int
	ocupados := 0
	for m := range 1440 {
		ocupados += variacao[m]
		lotados[m+1] = lotados[m]
		if ocupados >= c.Capacidade {
			lotados[m+1]++
		}
	}
	var inicios [1440]bool
	semana := (int(dia.Weekday())+6)%7 + 1
	for _, h := range c.Horarios {
		if h.DiaSemana != semana {
			continue
		}
		inicio, _ := minuto(h.Abre)
		fim, _ := minuto(h.Fecha)
		for m := inicio; m+duracao <= fim; m += duracao {
			if lotados[m+duracao] == lotados[m] && dia.Add(time.Duration(m)*time.Minute).After(agora) {
				inicios[m] = true
			}
		}
	}
	for m, livre := range inicios {
		if livre {
			vagas = append(vagas, relogio(m))
		}
	}
	return vagas
}

func (c Configuracao) horarios(dia time.Time, duracao int, agora time.Time) []string {
	if c.Intervalo <= 0 || duracao <= 0 {
		return []string{}
	}
	var inicios [1440]bool
	semana := (int(dia.Weekday())+6)%7 + 1
	for _, h := range c.Horarios {
		if h.DiaSemana != semana {
			continue
		}
		inicio, _ := minuto(h.Abre)
		fim, _ := minuto(h.Fecha)
		for m := inicio; m+duracao <= fim; m += duracao {
			if dia.Add(time.Duration(m) * time.Minute).After(agora) {
				inicios[m] = true
			}
		}
	}
	lista := []string{}
	for m, incluido := range inicios {
		if incluido {
			lista = append(lista, relogio(m))
		}
	}
	return lista
}

func idPositivo(s string) (int, bool) {
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err == nil && n > 0
}
