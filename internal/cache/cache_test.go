package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestLeitorCanceladoNaoBloqueiaNemDuplicaCalculo(t *testing.T) {
	c := New[int](time.Minute)
	iniciou, continuar := make(chan struct{}), make(chan struct{})
	var chamadas atomic.Int32
	calcular := func() (int, error) { chamadas.Add(1); close(iniciou); <-continuar; return 42, nil }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fim := make(chan error, 1)
	go func() { _, err := c.ObterContexto(ctx, "conta", calcular); fim <- err }()
	<-iniciou
	cancel()
	select {
	case err := <-fim:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("erro inesperado: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("leitor ficou retido após cancelamento")
	}
	close(continuar)
	valor, err := c.ObterContexto(context.Background(), "conta", func() (int, error) { t.Error("cálculo duplicado"); return 0, nil })
	if err != nil || valor != 42 || chamadas.Load() != 1 {
		t.Fatalf("resultado compartilhado inválido: %d, %v", valor, err)
	}
}

func TestContextoCanceladoNaoIniciaTrabalho(t *testing.T) {
	c := New[int](time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.ObterContexto(ctx, "x", func() (int, error) { t.Error("cálculo iniciou sem leitor ativo"); return 1, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelamento ignorado: %v", err)
	}
}

func TestFalhaDoCalculoNaoEncerraProcessoNemFicaEmCache(t *testing.T) {
	c := New[int](time.Minute)
	_, err := c.Obter("conta", func() (int, error) { panic("dado privado") })
	if !errors.Is(err, ErrCalculo) {
		t.Fatalf("falha inesperada: %v", err)
	}
	valor, err := c.Obter("conta", func() (int, error) { return 7, nil })
	if err != nil || valor != 7 {
		t.Fatal("cache não se recuperou da falha")
	}
}
