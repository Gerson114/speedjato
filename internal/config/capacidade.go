package config

import (
	"os"
	"strconv"
	"time"
)

const (
	RequisicoesSimultaneas = 128
	RequisicoesPorIP       = 10
	RajadaPorIP            = 30
	ConexoesBanco          = 50
	ConexoesBancoOciosas   = 10
	PrazoRespostaHTTP      = 125 * time.Second
	PrazoRequisicaoHTTP    = 110 * time.Second
)

// LimiteConexoesBanco permite dividir o orçamento de conexões entre réplicas.
// Valores inválidos mantêm o padrão, evitando iniciar com um limite sem controle.
func LimiteConexoesBanco() int {
	valor, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	if err != nil || valor < 1 || valor > ConexoesBanco {
		return ConexoesBanco
	}
	return valor
}
