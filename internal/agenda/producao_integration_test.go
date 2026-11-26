package agenda

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	database "lava/external/db"
	"lava/internal/registro/auth"
	"lava/internal/service"
)

func TestProducaoPostgres(t *testing.T) {
	dsn := os.Getenv("LAVA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LAVA_TEST_DATABASE_URL ausente; execute scripts/testar-producao.sh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("teste_lava_%d", time.Now().UnixNano())
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("Use URL postgres:// no banco de teste")
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	second, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	// Simula a atualização de uma instalação anterior às migrações versionadas.
	baseline, err := os.ReadFile("../../external/db/migrations/001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(baseline))
	exec("INSERT INTO usuarios(id,username,email,password,liberado) VALUES(1,'Teste','teste@example.invalid','!',true)")
	exec("INSERT INTO veiculos(id,cliente_id,apelido,modelo,placa) VALUES(1,1,'Carro','Modelo','AAA0001')")
	exec("INSERT INTO agenda_servicos(id,nome,duracao,preco_centavos) VALUES(1,'Lavagem',60,5000)")
	dia := time.Now().UTC().AddDate(0, 0, 2).Truncate(24 * time.Hour)
	exec(`INSERT INTO agendamentos(cliente_id,servico_id,veiculo_id,nome_servico,dia,inicio,fim,status)
 VALUES(1,1,1,'Lavagem',$1,600,660,'cancelado')`, dia.Format(time.DateOnly))
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, conn := range []*sql.DB{db, second} {
		wg.Add(1)
		go func(conn *sql.DB) { defer wg.Done(); results <- database.Migrar(ctx, conn) }(conn)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := database.VerificarMigracoes(ctx, db); err != nil {
		t.Fatal(err)
	}
	weekday := (int(dia.Weekday())+6)%7 + 1
	exec("UPDATE agenda_config SET capacidade=1,intervalo=60,horarios=$1", fmt.Sprintf(`[{"diaSemana":%d,"abre":"08:00","fecha":"18:00"}]`, weekday))
	a := api{db: db, local: time.UTC}
	b := api{db: second, local: time.UTC}
	before, err := b.versaoAgenda(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, nova, err := a.confirmar(ctx, 1, 1, 1, dia, "10:00")
	if err != nil || !nova {
		t.Fatalf("reagendar: %v, nova=%v", err, nova)
	}
	var cancelados int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM agendamentos WHERE status='cancelado'").Scan(&cancelados); err != nil || cancelados != 1 {
		t.Fatalf("histórico perdido: %d, %v", cancelados, err)
	}
	after, err := b.versaoAgenda(ctx)
	if err != nil || after == before {
		t.Fatalf("outra instância não recebeu alteração: %v", err)
	}
	// Repetições concorrentes do mesmo pedido retornam a reserva existente.
	ids := make(chan int64, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _, err := a.confirmar(ctx, 1, 1, 1, dia, "10:00")
			ids <- r.ID
			errs <- err
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for id := range ids {
		if id != r.ID {
			t.Fatal("pedido duplicado")
		}
	}
	exec("INSERT INTO veiculos(id,cliente_id,apelido,modelo,placa) VALUES(2,1,'Outro','Modelo','AAA0002')")
	if _, _, err := b.confirmar(ctx, 1, 1, 2, dia, "10:00"); err != errOcupado {
		t.Fatalf("capacidade excedida: %v", err)
	}
	s := service.Agendamentos{DB: db}
	if err := s.AtualizarLavagem(ctx, int(r.ID), "concluida"); err != service.ErrIndisponivel {
		t.Fatalf("lavagem pendente alterada: %v", err)
	}
	if err := s.AtualizarStatus(ctx, int(r.ID), "confirmado"); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE agendamentos SET chegou=true WHERE id=$1", r.ID)
	if err := s.AtualizarLavagem(ctx, int(r.ID), "proximo"); err != nil {
		t.Fatal(err)
	}
	outro, _, err := a.confirmar(ctx, 1, 1, 2, dia, "11:00")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AtualizarStatus(ctx, int(outro.ID), "confirmado"); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE agendamentos SET chegou=true WHERE id=$1", outro.ID)
	if err := s.AtualizarLavagem(ctx, int(outro.ID), "proximo"); err != service.ErrProximo {
		t.Fatalf("dois próximos permitidos: %v", err)
	}
	if err := s.AtualizarLavagem(ctx, int(r.ID), "concluida"); err != nil {
		t.Fatal(err)
	}
	if err := s.AtualizarStatus(ctx, int(r.ID), "cancelado"); err != service.ErrIndisponivel {
		t.Fatalf("concluído alterado: %v", err)
	}
	// Dois veículos disputando a última vaga devem produzir apenas uma reserva.
	disputa := make(chan error, 2)
	iniciar := make(chan struct{})
	for _, veiculo := range []int64{1, 2} {
		wg.Add(1)
		go func(veiculo int64) {
			defer wg.Done()
			<-iniciar
			_, _, err := a.confirmar(ctx, 1, 1, veiculo, dia, "13:00")
			disputa <- err
		}(veiculo)
	}
	close(iniciar)
	wg.Wait()
	close(disputa)
	vencedores := 0
	for err := range disputa {
		if err == nil {
			vencedores++
		} else if err != errOcupado {
			t.Fatal(err)
		}
	}
	if vencedores != 1 {
		t.Fatalf("última vaga reservada %d vezes", vencedores)
	}
	// Consultas repetidas e simultâneas criam apenas um lembrete por reserva.
	proxima := time.Now().UTC().Add(2 * time.Hour)
	if proxima.Hour()*60+proxima.Minute() > 1410 {
		proxima = time.Date(proxima.Year(), proxima.Month(), proxima.Day()+1, 0, 0, 0, 0, time.UTC)
	}
	var reservaLembrete int64
	err = db.QueryRowContext(ctx, `INSERT INTO agendamentos(cliente_id,servico_id,veiculo_id,nome_servico,dia,inicio,fim,status)
		VALUES(1,1,1,'Lavagem',$1,$2,$2+30,'confirmado') RETURNING id`, proxima.Format(time.DateOnly), proxima.Hour()*60+proxima.Minute()).Scan(&reservaLembrete)
	if err != nil {
		t.Fatal(err)
	}
	routerAvisos := gin.New()
	routerAvisos.GET("/avisos", func(c *gin.Context) { c.Set(auth.ContextUserID, 1); a.avisos(c) })
	respostas := make(chan int, 5)
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			routerAvisos.ServeHTTP(w, httptest.NewRequest("GET", "/avisos", nil))
			respostas <- w.Code
		}()
	}
	wg.Wait()
	close(respostas)
	for status := range respostas {
		if status != 200 {
			t.Fatalf("consulta de avisos: HTTP %d", status)
		}
	}
	var lembretes int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM avisos_cliente WHERE agendamento_id=$1 AND tipo='lembrete_24h'", reservaLembrete).Scan(&lembretes); err != nil || lembretes != 1 {
		t.Fatalf("lembretes criados: %d, erro: %v", lembretes, err)
	}
	// Sincroniza sequências após os IDs explícitos usados pelas fixtures.
	exec("SELECT setval(pg_get_serial_sequence('usuarios','id'),(SELECT max(id) FROM usuarios))")
	exec("SELECT setval(pg_get_serial_sequence('veiculos','id'),(SELECT max(id) FROM veiculos))")
	// Falha no aviso deve desfazer cliente, veículo, reserva e versão da agenda.
	exec(`CREATE FUNCTION falhar_aviso() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'falha de teste'; END; $$ LANGUAGE plpgsql;
 CREATE TRIGGER falhar_aviso BEFORE INSERT ON avisos_cliente FOR EACH ROW EXECUTE FUNCTION falhar_aviso()`)
	before, err = b.versaoAgenda(ctx)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"nome":"Novo","email":"novo@example.invalid","modelo":"Teste","placa":"BBB0001","servicoId":1,"data":%q,"inicio":"12:00"}`, dia.Format(time.DateOnly))
	router := gin.New()
	router.POST("/balcao", a.agendarBalcao)
	request := httptest.NewRequest("POST", "/balcao", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code < 400 {
		t.Fatal("falha de gravação ignorada")
	}
	var quantidade int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM usuarios WHERE email='novo@example.invalid'").Scan(&quantidade); err != nil || quantidade != 0 {
		t.Fatalf("gravação parcial: %d, %v", quantidade, err)
	}
	after, err = b.versaoAgenda(ctx)
	if err != nil || after != before {
		t.Fatal("rollback publicou evento")
	}

	t.Run("cinco envios do mesmo serviço", func(t *testing.T) {
		exec("SELECT setval(pg_get_serial_sequence('agenda_servicos','id'),(SELECT max(id) FROM agenda_servicos))")
		r := gin.New()
		r.POST("/servicos", a.salvarServico)
		status := make(chan int, 5)
		for range 5 {
			go func() {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/servicos", strings.NewReader(`{"nome":"Polimento","duracaoMinutos":30,"chaveCriacao":"formulario-polimento"}`))
				req.Header.Set("Content-Type", "application/json")
				r.ServeHTTP(w, req)
				status <- w.Code
			}()
		}
		criados := 0
		for range 5 {
			switch codigo := <-status; codigo {
			case 201:
				criados++
			case 200:
			default:
				t.Errorf("HTTP inesperado: %d", codigo)
			}
		}
		var total int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM agenda_servicos WHERE nome='Polimento'").Scan(&total); err != nil || total != 1 || criados != 1 {
			t.Fatalf("serviços=%d respostas de criação=%d erro=%v", total, criados, err)
		}
	})

	t.Run("excluir serviço com agendamento", func(t *testing.T) {
		exec("INSERT INTO estoque_produtos(id,nome,unidade,quantidade) VALUES(9123,'Produto de teste','ml',100)")
		exec("INSERT INTO servico_produtos(servico_id,produto_id) VALUES(1,9123)")

		r := gin.New()
		r.DELETE("/servicos/:id", a.excluirServico)
		r.GET("/servicos", a.servicos)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("DELETE", "/servicos/1", nil))
		if w.Code != 204 {
			t.Fatalf("exclusão: HTTP %d", w.Code)
		}
		exec("DELETE FROM estoque_produtos WHERE id=9123")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/servicos", nil))
		var catalogo []Servico
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &catalogo) != nil {
			t.Fatal("resposta de catálogo inválida")
		}
		for _, s := range catalogo {
			if s.ID == 1 {
				t.Fatal("serviço excluído ainda no catálogo")
			}
		}
		var historico bool
		if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM agendamentos WHERE servico_id=1)").Scan(&historico); err != nil || !historico {
			t.Fatalf("histórico perdido: %v", err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := lerServico(ctx, tx, 1, true); err != sql.ErrNoRows {
			t.Fatalf("serviço disponível para reserva: %v", err)
		}
	})
}
