"use client";
import { acompanhar as acompanharDados } from "../acompanhar";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  api,
  Falha,
  Icone,
  Marca,
  mensagem,
  type Requisitar,
  type Reserva,
} from "./base";
import { deslocarDia, etapas } from "./agenda/agenda-util";
import { Servicos, Funcionamento } from "./configuracao/editores";
import Estoque from "./estoque/estoque";
import { AgendarBalcao } from "./agenda/agendar-balcao";

function Login({ entrar }: { entrar: () => void }) {
  const [mostrar, setMostrar] = useState(false);
  const [erro, setErro] = useState("");
  const [ocupado, setOcupado] = useState(false);
  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (ocupado) return;
    const dados = new FormData(e.currentTarget);
    setOcupado(true);
    setErro("");
    try {
      await api("login", "POST", {
        email: String(dados.get("email")).trim(),
        password: dados.get("senha"),
      });
      entrar();
    } catch (e) {
      setErro(mensagem(e));
    } finally {
      setOcupado(false);
    }
  }
  return (
    <main className="login">
      <section className="login-lado">
        <header className="identidade">
          <Marca />
          <span className="divisor" />
          <span>Administração</span>
        </header>
        <div className="acesso">
          <div className="icone-bloco">
            <Icone nome="lock" />
          </div>
          <h1>
            Acesso do
            <br />
            administrador<span>.</span>
          </h1>
          <form onSubmit={enviar}>
            <label>
              E-mail
              <input
                type="email"
                name="email"
                placeholder="seu@email.com"
                autoComplete="username"
                required
                maxLength={254}
              />
            </label>
            <label htmlFor="senha">Senha</label>
            <div className="senha">
              <input
                id="senha"
                type={mostrar ? "text" : "password"}
                name="senha"
                placeholder="Digite sua senha"
                autoComplete="current-password"
                required
              />
              <button
                type="button"
                className="icone"
                onClick={() => setMostrar(!mostrar)}
                aria-label={mostrar ? "Ocultar senha" : "Mostrar senha"}
                aria-pressed={mostrar}
              >
                <Icone nome={mostrar ? "hidden" : "eye"} />
              </button>
            </div>
            {erro && (
              <p className="erro" role="alert">
                {erro}
              </p>
            )}
            <button className="primario grande" disabled={ocupado} style={{ marginTop: 24 }}>
              {ocupado ? "Entrando…" : "Entrar no painel"}
            </button>
          </form>
        </div>
      </section>
      <div className="foto" aria-hidden="true" />
    </main>
  );
}

function hoje() {
  return new Intl.DateTimeFormat("sv-SE", {
    timeZone: "America/Fortaleza",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}

const indicadores = {
  carrosAtendidos: "Carros atendidos",
  cancelados: "Agendamentos cancelados",
  pendentes: "Pendentes de confirmação",
  aguardando: "Confirmados aguardando lavagem",
  proximos: "Próximos na fila",
  emLavagem: "Em lavagem",
} as const;
type Resumo = Record<keyof typeof indicadores, number> & { totalCentavos: number };
type Relatorios = {servicosMaisVendidos:{servico:string;quantidade:number;totalCentavos:number}[];clientesRecorrentes:{cliente:string;quantidade:number;totalCentavos:number}[]};
function Dashboard({ requisitar }: { requisitar: Requisitar }) {
  const [periodo,setPeriodo]=useState("hoje");
  const [resumo,setResumo]=useState<Resumo|null>(null);
  const [relatorios,setRelatorios]=useState<Relatorios|null>(null);
  const [erro,setErro]=useState("");
  useEffect(() => {
    const params = new URLSearchParams({periodo});
    return acompanharDados(signal => Promise.all([
      requisitar<Resumo>("resumo?"+params,"GET",undefined,signal),
      requisitar<Relatorios>("relatorios?"+params,"GET",undefined,signal),
    ]), ([r,l]) => { setResumo(r); setRelatorios(l); setErro(""); }, e => setErro(mensagem(e)));
  }, [periodo,requisitar]);
  return <section className="dash-resumo"><div className="dash-cabecalho"><div><p className="dash-etiqueta">VISÃO GERAL</p><h2>Seu movimento</h2></div><select aria-label="Período do resumo" value={periodo} onChange={e=>{setErro("");setResumo(null);setRelatorios(null);setPeriodo(e.target.value)}}><option value="hoje">Hoje</option><option value="7dias">Últimos 7 dias</option><option value="mes">Este mês</option></select></div>{erro?<p className="erro">{erro}</p>:<><div className="dash-cartoes">{(Object.keys(indicadores) as (keyof typeof indicadores)[]).map(chave => (
    <article key={chave}><span>{indicadores[chave]}</span><strong>{resumo?.[chave] ?? "—"}</strong></article>
  ))}<article><span>Faturamento previsto</span><strong>{resumo? (resumo.totalCentavos/100).toLocaleString("pt-BR",{style:"currency",currency:"BRL"}):"—"}</strong><small>reservas não canceladas</small></article></div><div className="relatorio-grids"><article><h3>Serviços mais vendidos</h3>{!relatorios?<p>Carregando…</p>:relatorios.servicosMaisVendidos.length?relatorios.servicosMaisVendidos.map(x=><p key={x.servico}><span>{x.servico}</span><strong>{x.quantidade} · {(x.totalCentavos/100).toLocaleString("pt-BR",{style:"currency",currency:"BRL"})}</strong></p>):<p>Sem vendas no período.</p>}</article><article><h3>Clientes recorrentes</h3>{!relatorios?<p>Carregando…</p>:relatorios.clientesRecorrentes.length?relatorios.clientesRecorrentes.map(x=><p key={x.cliente}><span>{x.cliente}</span><strong>{x.quantidade} visitas</strong></p>):<p>Ainda não há clientes com mais de um agendamento.</p>}</article></div></>}</section>;
}

function Agenda({ requisitar, historico = false }: { requisitar: Requisitar; historico?: boolean }) {
  const [categoria, setCategoria] = useState("concluidos");
  const [busca, setBusca] = useState("");
  const [pagina, setPagina] = useState(1);
  const [data, setData] = useState(hoje);
  const consulta = new URLSearchParams(historico ? {visao: "historico", categoria, busca, pagina: String(pagina)} : {visao: "ativos", data, pagina: String(pagina)}).toString();
  const [cache, setCache] = useState(() => new Map<string, Reserva[]>());
  const reservas = cache.get(consulta);
  const [erro, setErro] = useState("");
  const [erroConsulta, setErroConsulta] = useState("");
  const [versao, setVersao] = useState(0);
  const [abrirAgendamento,setAbrirAgendamento]=useState(false);
  const [selecionada,setSelecionada]=useState<Reserva|null>(null);
  const [salvandoStatus,setSalvandoStatus]=useState(false);
  const [confirmarCancelamento, setConfirmarCancelamento] = useState(false);
  useEffect(() => {
    let versaoExibida: string | undefined;
    return acompanharDados(async signal => {
      let versaoAtual: string | undefined;
      try {
        const resposta = await requisitar<{versao:string}>(
          "agendamentos/novidades?versao=" + encodeURIComponent(versaoExibida ?? ""),
          "GET", undefined, signal);
        versaoAtual = resposta.versao;
        if (versaoAtual !== undefined && versaoAtual === versaoExibida) return null;
      } catch (e) {
        if (signal.aborted) throw e;
        // A lista continua disponível quando a conexão de novidades falha.
      }
      const lista = await requisitar<Reserva[]>("agendamentos?" + consulta, "GET", undefined, signal);
      return {lista, versaoAtual};
    }, resultado => {
      if (resultado) {
        versaoExibida = resultado.versaoAtual;
        setCache(anterior => {
          const novo = new Map(anterior);
          novo.delete(consulta);
          novo.set(consulta, resultado.lista);
          if (novo.size > 20) novo.delete(novo.keys().next().value!);
          return novo;
        });
        setSelecionada(atual => resultado.lista.find(r => r.id === atual?.id) ?? null);
      }
      setErroConsulta("");
    }, e => setErroConsulta(mensagem(e)), 1000);
  }, [consulta, versao, requisitar]);
  function atualizar(dia: string) {
    if (!dia) return;
    setErro("");
    setPagina(1);
    setData(dia);
    setVersao((v) => v + 1);
  }
  async function alterarStatus(status: "confirmado" | "cancelado", reserva = selecionada) {
    if (!reserva || salvandoStatus) return;
    setSalvandoStatus(true); setErro("");
    try {
      await requisitar(`agendamentos/${reserva.id}/status`, "PUT", { status });
      setSelecionada(null);
      setVersao(v => v + 1);
    } catch (e) { setErro(mensagem(e)); }
    finally { setSalvandoStatus(false); }
  }
  async function alterarLavagem(status: Reserva["lavagemStatus"]) {
    if (!selecionada) return;
    setSalvandoStatus(true); setErro("");
    try { await requisitar(`agendamentos/${selecionada.id}/lavagem`, "PUT", {status}); setSelecionada(null); setVersao(v=>v+1); }
    catch(e) { setErro(mensagem(e)); }
    finally { setSalvandoStatus(false); }
  }
  async function registrarChegada() {
    if (!selecionada || salvandoStatus) return;
    setSalvandoStatus(true); setErro("");
    try {
      await requisitar(`agendamentos/${selecionada.id}/chegada`, "PUT", {});
      setSelecionada({...selecionada, chegou: true});
      setVersao(v => v + 1);
    } catch(e) { setErro(mensagem(e)); }
    finally { setSalvandoStatus(false); }
  }
  const acoes = selecionada?.acoes;
  const etapasDisponiveis = selecionada?.chegou ? acoes?.etapas ?? [] : [];
  return (
    <section>
      <div className="titulo">
        <div>
          <h1>{historico ? "Histórico de atendimentos" : "Agenda do dia"}</h1>
        </div>
        {historico && <label>Atendimentos<select value={categoria} onChange={e => { setCategoria(e.target.value); setPagina(1); }}><option value="concluidos">Concluídos</option><option value="cancelados">Cancelados</option></select></label>}
        {historico ? <label>Pesquisar no histórico<input type="search" placeholder="Cliente, placa, modelo ou serviço" value={busca} onChange={e => {setBusca(e.target.value); setPagina(1); }} /></label> : <div className="acoes-agenda"><button className="primario" onClick={() => setAbrirAgendamento(true)}>+ Agendar</button><div className="navegar-dia">
          <label htmlFor="data-agenda">Data da agenda</label>
          <div>
            <button className="secundario" aria-label="Dia anterior" onClick={() => atualizar(deslocarDia(data, -1))}>‹</button>
            <input id="data-agenda" type="date" value={data} onChange={e => atualizar(e.target.value)} />
            <button className="secundario" aria-label="Próximo dia" onClick={() => atualizar(deslocarDia(data, 1))}>›</button>
            <button className="texto" onClick={() => atualizar(hoje())}>Hoje</button>
          </div>
        </div></div>}
      </div>
      <div className="quadro">
        {(erro || erroConsulta) && (
          <div className="vazio" role="alert">
            {erro || erroConsulta}
          </div>
        )}
        {!reservas ? (
          <div className="vazio" role="status">
            Carregando reservas…
          </div>
        ) : reservas.length === 0 ? (
          <div className="vazio">
            <div className="icone-bloco">
              <Icone nome="agenda" />
            </div>
            <h3>{historico ? "Nenhum atendimento encontrado" : "Nenhum agendamento nesta data"}</h3>
            <p>{historico ? (busca ? "Tente outro termo de pesquisa." : `Atendimentos ${categoria === "concluidos" ? "concluídos" : "cancelados"} aparecerão aqui.`) : "Novos atendimentos aparecerão aqui."}</p>
          </div>
        ) : (
          <div className="agenda-cartoes">
            {reservas.slice(0,50).map(r => (
              <article key={r.id} className="agenda-cartao" data-status={r.status} data-lavagem={historico || r.status === "cancelado" ? undefined : r.lavagemStatus}>
                <span className="agenda-cartao-topo"><span className="agenda-hora">{historico && r.data.split("-").reverse().join("/") + " · "}{r.inicio} <small>– {r.fim}</small></span><span className={`status-agenda status-${r.status}`}>{historico ? (r.status === "cancelado" ? "Cancelado" : "Concluído") : r.status === "pendente" ? "Pendente" : r.status === "confirmado" ? "Confirmado" : "Cancelado"}</span></span>
                <span className="agenda-cliente">{r.cliente}</span>
                <span className="agenda-veiculo">{r.modelo || r.veiculo || "Veículo não informado"}{r.placa ? ` · ${r.placa}` : ""}</span>
                <span className="agenda-cartao-rodape"><span>{r.servico}</span><span>{r.status === "cancelado" ? "Cancelada" : etapas[r.lavagemStatus]}</span></span>
                <div className="agenda-acoes-cartao">
                  {r.acoes?.confirmar && (
                    <button type="button" className="primario" disabled={salvandoStatus}
                      onClick={() => alterarStatus("confirmado", r)}>
                      Confirmar agendamento
                    </button>
                  )}
                  {r.status === "confirmado" && <span className="aviso-confirmado">✓ Agendamento confirmado</span>}
                  <button type="button" className="texto abrir-cartao" onClick={() => setSelecionada(r)}
                    aria-label={`Ver detalhes do agendamento de ${r.cliente}`}>Ver detalhes</button>
                </div>
              </article>
            ))}
          </div>
        )}
      </div>
      {reservas && (pagina > 1 || reservas.length > 50) && <div className="acoes"><button type="button" className="secundario" disabled={pagina === 1} onClick={() => {setPagina(pagina-1)}}>Anterior</button><span>Página {pagina}</span><button type="button" className="secundario" disabled={reservas.length <= 50} onClick={() => {setPagina(pagina+1)}}>Próxima</button></div>}
      {selecionada && <div className="modal-fundo" onMouseDown={e => { if (e.target === e.currentTarget) setSelecionada(null); }}><section className="modal-detalhes" role="dialog" aria-modal="true" aria-labelledby="titulo-detalhes"><div className="modal-topo"><div><p className="dash-etiqueta">AGENDAMENTO #{selecionada.id}</p><h2 id="titulo-detalhes">Detalhes da reserva</h2></div><button className="texto" onClick={() => setSelecionada(null)} aria-label="Fechar detalhes">Fechar</button></div><dl className="detalhes-grid"><div><dt>Cliente</dt><dd>{selecionada.cliente}</dd></div><div><dt>E-mail</dt><dd>{selecionada.email || "Não informado"}</dd></div><div><dt>Veículo</dt><dd>{selecionada.modelo || selecionada.veiculo}</dd></div><div><dt>Placa</dt><dd>{selecionada.placa || "Não informada"}</dd></div><div><dt>Cor</dt><dd>{selecionada.cor || "Não informada"}</dd></div><div><dt>Serviço</dt><dd>{selecionada.servico}</dd></div><div><dt>Dia e horário</dt><dd>{selecionada.data} · {selecionada.inicio}–{selecionada.fim}</dd></div><div><dt>Preço</dt><dd>{(selecionada.precoCentavos/100).toLocaleString("pt-BR",{style:"currency",currency:"BRL"})}</dd></div><div><dt>Reserva</dt><dd>{selecionada.status}</dd></div><div><dt>Chegada</dt><dd>{selecionada.chegou ? "Cliente chegou" : selecionada.status === "confirmado" ? <button className="primario" disabled={salvandoStatus} onClick={registrarChegada}>Marcar que chegou</button> : "Aguardando confirmação"}</dd></div><div><dt>Andamento da lavagem</dt><dd>{etapasDisponiveis.length ? (
                <div className="etapas-lavagem" role="group" aria-label="Andamento da lavagem">
                  {etapasDisponiveis.map(etapa => (
                    <button key={etapa} type="button" className="secundario"
                      disabled={salvandoStatus || etapa === selecionada.lavagemStatus}
                      aria-pressed={etapa === selecionada.lavagemStatus}
                      onClick={() => alterarLavagem(etapa)}>
                      {etapa === "em_andamento" ? "Em lavagem · vez dele" : etapas[etapa]}
                    </button>
                  ))}
                </div>
              ) : selecionada.status === "cancelado" ? "Cancelada" : !selecionada.chegou && selecionada.status === "confirmado" ? "Registre a chegada para alterar o andamento" : etapas[selecionada.lavagemStatus]}</dd></div></dl>{!acoes && <p role="status">A API não informou as ações disponíveis para este agendamento.</p>}{erro && <p className="erro" role="alert">{erro}</p>}<div className="modal-acoes"><button className="secundario" disabled={salvandoStatus || !acoes?.cancelar} onClick={() => setConfirmarCancelamento(true)}>{salvandoStatus ? "Salvando…" : "Cancelar agendamento"}</button><button className="primario" disabled={salvandoStatus || !acoes?.confirmar} onClick={() => alterarStatus("confirmado")}>{selecionada.status === "confirmado" ? "Agendamento confirmado" : "Confirmar agendamento"}</button></div></section></div>}
      {confirmarCancelamento && selecionada && <div className="modal-fundo" onMouseDown={e => { if (e.target === e.currentTarget && !salvandoStatus) setConfirmarCancelamento(false); }}><section className="confirmar-exclusao" role="alertdialog" aria-modal="true" aria-labelledby="cancelar-agendamento-titulo"><h2 id="cancelar-agendamento-titulo">Cancelar agendamento?</h2><p>Deseja cancelar o agendamento de {selecionada.cliente}?</p><div className="modal-acoes"><button className="secundario" disabled={salvandoStatus} onClick={() => setConfirmarCancelamento(false)}>Voltar</button><button className="primario" disabled={salvandoStatus} onClick={async () => { await alterarStatus("cancelado"); setConfirmarCancelamento(false); }}>{salvandoStatus ? "Cancelando…" : "Sim, cancelar agendamento"}</button></div></section></div>}
      {abrirAgendamento && <AgendarBalcao diaInicial={data} requisitar={requisitar} fechar={() => setAbrirAgendamento(false)} concluido={dia => { setAbrirAgendamento(false); atualizar(dia); }} />}
    </section>
  );
}

const abas = [
  { id: "agenda", rotulo: "Agenda do dia", icone: "agenda" },
  { id: "historico", rotulo: "Histórico", icone: "historico" },
  { id: "visao", rotulo: "Visão geral", icone: "visao" },
  { id: "servicos", rotulo: "Serviços", icone: "servicos" },
  { id: "config", rotulo: "Funcionamento", icone: "config" },
  { id: "estoque", rotulo: "Estoque", icone: "estoque" },
] as const;
function secaoAtual() {
  return abas.find(({ id }) => "#" + id === window.location.hash)?.id ?? "agenda";
}

export default function Painel({ comCookie }: { comCookie: boolean }) {
  const [logado, setLogado] = useState(comCookie);
  const [aba, setAba] = useState<(typeof abas)[number]["id"]>("agenda");
  useEffect(() => {
    const sincronizar = () => setAba(secaoAtual());
    sincronizar();
    window.addEventListener("hashchange", sincronizar);
    window.addEventListener("popstate", sincronizar);
    return () => {
      window.removeEventListener("hashchange", sincronizar);
      window.removeEventListener("popstate", sincronizar);
    };
  }, []);
  const [erro, setErro] = useState("");
  const requisitar: Requisitar = useCallback(
    async <T,>(
      path: string,
      method?: string,
      body?: unknown,
      signal?: AbortSignal,
    ) => {
      try {
        return await api<T>(path, method, body, signal);
      } catch (e) {
        if (e instanceof Falha && e.status === 401 && !signal?.aborted)
          setLogado(false);
        throw e;
      }
    },
    [],
  );
  async function sair() {
    try {
      await api("sair", "POST");
      setErro("");
      setLogado(false);
    } catch (e) {
      setErro(mensagem(e));
    }
  }
  if (!logado)
    return (
      <Login
        entrar={() => {
          setLogado(true);
          setErro("");
        }}
      />
    );
  return (
    <div className="painel">
      <header className="barra">
        <div className="largura barra-dentro">
          <div className="identidade">
            <Marca />
            <span className="divisor" />
            <span>Administração</span>
          </div>
          <button className="sair" onClick={sair}>
            <Icone nome="sair" />
            Sair
          </button>
        </div>
      </header>
      <nav className="navegacao-admin" aria-label="Navegação administrativa">
        <div className="menu-secoes">
          {abas.map((a) => (
            <a
              key={a.id}
              aria-current={aba === a.id ? "page" : undefined}
              href={`#${a.id}`}
              onClick={(evento) => {
                if (evento.metaKey || evento.ctrlKey || evento.shiftKey || evento.altKey) return;
                setAba(a.id);
              }}
            >
              <Icone nome={a.icone} />
              <span>{a.rotulo}</span>
            </a>
          ))}
        </div>
      </nav>
      <main className="largura conteudo">
        {erro && (
          <p className="erro" role="alert">
            {erro}
          </p>
        )}
        {aba === "agenda" && <Agenda requisitar={requisitar} />}
        {aba === "historico" && <Agenda historico requisitar={requisitar} />}
        {aba === "visao" && <Dashboard requisitar={requisitar} />}
        {aba === "servicos" && <Servicos requisitar={requisitar} />}
        {aba === "config" && <Funcionamento requisitar={requisitar} />}
        {aba === "estoque" && <Estoque requisitar={requisitar} />}
      </main>
    </div>
  );
}
