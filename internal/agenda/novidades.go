package agenda

import (
	"context"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

func (a api) versaoAgenda(ctx context.Context) (string, error) {
	var versao int64
	err := a.db.QueryRowContext(ctx, "SELECT versao FROM agenda_versao WHERE id=1").Scan(&versao)
	return strconv.FormatInt(versao, 10), err
}

func (a api) aguardarPedido(c *gin.Context) {
	timer := time.NewTimer(14 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		versao, err := a.versaoAgenda(c.Request.Context())
		if err != nil {
			falha(c, err)
			return
		}
		if c.Query("versao") != versao {
			c.Header("Cache-Control", "no-store")
			c.JSON(200, gin.H{"versao": versao})
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-timer.C:
			c.Header("Cache-Control", "no-store")
			c.JSON(200, gin.H{"versao": versao})
			return
		case <-ticker.C:
		}
	}
}
