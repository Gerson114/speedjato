// Package cache oferece um cache em memória com validade (TTL) e deduplicação
// de chamadas simultâneas (singleflight).

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var ErrCalculo = errors.New("não foi possível calcular os dados em cache")

// entrada é um valor guardado com o instante em que expira.
type entrada[V any] struct {
	valor    V
	expiraEm time.Time
	tamanho  int
}

// Cache guarda valores do tipo V por chave string, cada um válido por ttl.
// É seguro para uso concorrente.
type Cache[V any] struct {
	ttl         time.Duration
	mu          sync.Mutex
	m           map[string]entrada[V]
	sf          singleflight.Group
	maxEntradas int
	bytes       int
	maxBytes    int
}

// New cria um cache cujas entradas valem por ttl.
func New[V any](ttl time.Duration) *Cache[V] {
	return &Cache[V]{ttl: ttl, m: make(map[string]entrada[V]), maxEntradas: 512, maxBytes: 16 << 20}
}

// Obter devolve o valor da chave se ainda estiver válido; senão, chama calcular
// para produzi-lo, guarda e devolve. Chamadas concorrentes para a MESMA chave
// executam calcular uma única vez (as demais aguardam e reaproveitam o
// resultado). Se calcular falhar, nada é guardado e o erro é repassado.
func (c *Cache[V]) Obter(chave string, calcular func() (V, error)) (V, error) {
	return c.ObterContexto(context.Background(), chave, calcular)
}

// ObterContexto permite que cada leitor desista sem esperar o cálculo compartilhado.
func (c *Cache[V]) ObterContexto(ctx context.Context, chave string, calcular func() (V, error)) (V, error) {
	if err := ctx.Err(); err != nil {
		var zero V
		return zero, err
	}
	if v, ok := c.ler(chave); ok {
		return v, nil
	}

	resultado := c.sf.DoChan(chave, func() (valor any, err error) {
		// O cálculo agora roda fora da goroutine do handler. Um panic não pode
		// escapar da recuperação HTTP e encerrar o processo inteiro.
		defer func() {
			if recover() != nil {
				var zero V
				valor, err = zero, ErrCalculo
				// Chaves podem conter tokens; nem chave nem panic entram no log.
				slog.Error("falha inesperada ao calcular dados em cache")
			}
		}()
		// Recheca: outra goroutine pode ter populado enquanto esperávamos.
		if v, ok := c.ler(chave); ok {
			return v, nil
		}
		v, err := calcular()
		if err != nil {
			return v, err
		}
		c.guardar(chave, v)
		return v, nil
	})
	select {
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	case res := <-resultado:
		if res.Err != nil {
			var zero V
			return zero, res.Err
		}
		return res.Val.(V), nil
	}
}

// Invalidar remove uma chave do cache (força recálculo na próxima leitura).
func (c *Cache[V]) Invalidar(chave string) {
	c.mu.Lock()
	c.remover(chave)
	c.mu.Unlock()
}

func (c *Cache[V]) ler(chave string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[chave]
	if !ok || !time.Now().Before(e.expiraEm) {
		c.remover(chave)
		var zero V
		return zero, false
	}
	return e.valor, true
}

// remover is called with c.mu held.
func (c *Cache[V]) remover(chave string) {
	if e, ok := c.m[chave]; ok {
		c.bytes -= e.tamanho
		delete(c.m, chave)
	}
}

func (c *Cache[V]) guardar(chave string, v V) {
	// A count cap alone allows large chat histories to exhaust RAM. Bound encoded
	// payload sizes too; oversized results can be returned without retaining them.
	raw, err := json.Marshal(v)
	if err != nil || len(raw) > 1<<20 {
		return
	}
	tamanho := len(raw)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.remover(chave)
	if len(c.m) >= c.maxEntradas || c.bytes+tamanho > c.maxBytes {
		agora := time.Now()
		for k, e := range c.m {
			if !agora.Before(e.expiraEm) {
				c.remover(k)
			}
		}
		for len(c.m) > 0 && (len(c.m) >= c.maxEntradas || c.bytes+tamanho > c.maxBytes) {
			for k := range c.m {
				c.remover(k)
				break
			}
		}
	}
	c.m[chave] = entrada[V]{valor: v, expiraEm: time.Now().Add(c.ttl), tamanho: tamanho}
	c.bytes += tamanho
}
