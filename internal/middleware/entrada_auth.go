package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// ValidarEntradaAuth valida o corpo JSON e a seleção de painel.
func ValidarEntradaAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		q, err := url.ParseQuery(c.Request.URL.RawQuery)
		if err != nil || len(q["painel"]) > 1 || (q.Get("painel") != "" && q.Get("painel") != "admin" && q.Get("painel") != "user") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Filtros inválidos"})
			return
		}
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			// Enforce route limits before buffering or parsing, even when mounted
			// without the production LimiteCorpo middleware.
			const limite = 8 << 10
			if c.Request.ContentLength > limite {
				c.AbortWithStatusJSON(413, gin.H{"error": "Corpo muito grande"})
				return
			}
			body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, limite))
			if err != nil {
				c.AbortWithStatusJSON(413, gin.H{"error": "Corpo muito grande ou inválido"})
				return
			}
			if len(body) > 0 {
				media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
				if err != nil || media != "application/json" {
					c.AbortWithStatusJSON(415, gin.H{"error": "Use application/json"})
					return
				}
				if !utf8.Valid(body) {
					c.AbortWithStatusJSON(400, gin.H{"error": "JSON inválido"})
					return
				}
				dec := json.NewDecoder(bytes.NewReader(body))
				dec.UseNumber()
				if len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' {
					c.AbortWithStatusJSON(400, gin.H{"error": "Objeto JSON obrigatório"})
					return
				}
				err = validarJSON(dec, 0)
				if err == nil {
					_, err = dec.Token()
					if err == io.EOF {
						err = nil
					} else {
						err = fmt.Errorf("JSON adicional")
					}
				}
				if err != nil {
					c.AbortWithStatusJSON(400, gin.H{"error": "JSON inválido, repetido ou excessivo"})
					return
				}
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		c.Next()
	}
}

func validarJSON(d *json.Decoder, depth int) error {
	if depth > 16 {
		return fmt.Errorf("profundidade")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	if s, ok := t.(string); ok {
		if len(s) > 16384 || strings.ContainsRune(s, 0) {
			return fmt.Errorf("texto excessivo")
		}
		return nil
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return fmt.Errorf("delimitador")
	}
	seen := map[string]bool{}
	count := 0
	for d.More() {
		count++
		if count > 1024 {
			return fmt.Errorf("itens excessivos")
		}
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			k, ok := key.(string)
			if !ok || len(k) > 128 || seen[strings.ToLower(k)] {
				return fmt.Errorf("campo duplicado")
			}
			seen[strings.ToLower(k)] = true
		}
		if err := validarJSON(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}
