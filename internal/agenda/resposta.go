package agenda

import "lava/internal/service"

type itemAgenda struct {
	Reserva
	Acoes           service.AcoesAgenda `json:"acoes"`
	SituacaoCliente string              `json:"situacaoCliente"`
}

func apresentarReserva(r Reserva) itemAgenda {
	return itemAgenda{r, service.AcoesDisponiveis(r.Status, r.LavagemStatus), service.SituacaoCliente(r.Status, r.LavagemStatus)}
}
