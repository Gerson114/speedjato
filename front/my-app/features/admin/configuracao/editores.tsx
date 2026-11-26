"use client";

import { acompanhar } from "../../acompanhar";
import { useEffect, useRef, useState, type FormEvent } from "react";
import {
  Icone,
  mensagem,
  type Config,
  type Periodo,
  type Requisitar,
  type Servico,
} from "../base";

function useDados<T>(rota: string, requisitar: Requisitar, pausado = false) {
  const [dados, definir] = useState<T | null>(null);
  const [erro, setErro] = useState("");
  const [erroConsulta, setErroConsulta] = useState("");
  const [versao, setVersao] = useState(0);
  useEffect(() => {
    if (pausado) return;
    return acompanhar(signal => requisitar<T>(rota, "GET", undefined, signal),
      dados => { definir(dados); setErroConsulta(""); }, e => setErroConsulta(mensagem(e)));
  }, [rota, requisitar, versao, pausado]);
  return {
    dados,
    definir,
    erro: erro || erroConsulta,
    setErro,
    tentar: () => {
      setErro("");
      setVersao((n) => n + 1);
    },
  };
}

function Carregamento({ erro, tentar }: { erro: string; tentar: () => void }) {
  return (
    <div className="quadro vazio" role="status">
      {erro || "Carregando…"}
      {erro && (
        <button className="secundario" onClick={tentar}>
          Tentar novamente
        </button>
      )}
    </div>
  );
}

export function Servicos({ requisitar }: { requisitar: Requisitar }) {
  const { dados, definir, erro, setErro, tentar } = useDados<Servico[]>(
    "servicos",
    requisitar,
  );
  const [produtos, setProdutos] = useState<{id:number;nome:string}[] | null>(null);
  useEffect(() => {
    return acompanhar(signal => requisitar<{id:number;nome:string}[]>("estoque", "GET", undefined, signal),
      setProdutos, e => setErro(mensagem(e)));
  }, [requisitar, setErro]);
  const [edicao, setEdicao] = useState<Servico | null>(null);
  const [salvando, setSalvando] = useState(false);
  const [aviso, setAviso] = useState("");
  const enviando = useRef(false);
  const chaveCriacao = useRef("");
  const [exclusaoPendente, setExclusaoPendente] = useState<Servico | null>(null);
  function editar(s: Servico | null) {
    if (s && !s.id) {
      chaveCriacao.current = Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, "0")).join("");
    }
    setEdicao(s);
    setAviso("");
    setErro("");
  }
  async function excluir(s: Servico) {
    if (enviando.current) return;
    enviando.current = true;
    setSalvando(true);
    setErro("");
    setAviso("");
    try {
      await requisitar("servicos/" + s.id, "DELETE");
      definir(lista => lista?.filter(item => item.id !== s.id) ?? null);
      if (edicao?.id === s.id) setEdicao(null);
      setExclusaoPendente(null);
      setAviso("Serviço excluído do catálogo.");
    } catch (e) {
      setErro(mensagem(e));
    } finally {
      enviando.current = false;
      setSalvando(false);
    }
  }
  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!edicao || !dados || enviando.current) return;
    const f = new FormData(e.currentTarget);
    enviando.current = true;
    setSalvando(true);
    setErro("");
    setAviso("");
    try {
      const s = await requisitar<Servico>(
        edicao.id ? "servicos/" + edicao.id : "servicos",
        edicao.id ? "PUT" : "POST",
        {
          ...(!edicao.id && { chaveCriacao: chaveCriacao.current }),
          nome: String(f.get("nome")).trim(),
          duracaoMinutos: Number(f.get("duracao")),
          precoCentavos: Math.round(Number(f.get("preco")) * 100),
          descricao: String(f.get("descricao")).trim(),
          produtoIds: f.getAll("produtoIds").map(Number),
          itens: String(f.get("itens")).split("\n").map(item => item.trim()).filter(Boolean),
        },
      );
      definir(
        [...dados.filter((i) => i.id !== s.id), s].sort((a, b) =>
          a.nome.localeCompare(b.nome, "pt-BR"),
        ),
      );
      setEdicao(null);
      setAviso("Serviço salvo.");
    } catch (e) {
      setErro(mensagem(e));
    } finally {
      enviando.current = false;
      setSalvando(false);
    }
  }
  return (
    <section>
      {exclusaoPendente && <div className="modal-fundo" onMouseDown={e => {if (e.target === e.currentTarget && !salvando) setExclusaoPendente(null)}}><section className="confirmar-exclusao confirmar-servico" role="alertdialog" aria-modal="true" aria-labelledby="confirmar-exclusao-titulo" aria-describedby="confirmar-exclusao-descricao"><div className="confirmar-servico-icone" aria-hidden="true"><Icone nome="servicos" /></div><h2 id="confirmar-exclusao-titulo">Excluir serviço?</h2><p id="confirmar-exclusao-descricao">O serviço <strong>{exclusaoPendente.nome}</strong> será retirado do catálogo.</p><p className="confirmar-servico-nota">Os agendamentos existentes continuam no histórico.</p><div className="modal-acoes"><button type="button" className="secundario" disabled={salvando} onClick={() => setExclusaoPendente(null)}>Cancelar</button><button type="button" className="excluir-produto" disabled={salvando} onClick={() => void excluir(exclusaoPendente)}>{salvando ? "Excluindo…" : "Excluir serviço"}</button></div></section></div>}
      <div className="titulo">
        <div>
          <h1>Serviços</h1>
          <p>Defina quais lavagens o cliente pode agendar.</p>
        </div>
        {dados && !edicao && (
          <button
            className="primario"
            onClick={() => editar({ id: 0, nome: "", duracaoMinutos: 30, precoCentavos: 0, descricao: "", itens: [], produtoIds: [], produtos: [] })}
          >
            <Icone nome="plus" />
            Novo serviço
          </button>
        )}
      </div>
      {edicao && (
        <form className="quadro editor" key={edicao.id} onSubmit={salvar}>
          <h2>{edicao.id ? "Editar serviço" : "Novo serviço"}</h2>
          <div className="linha">
            <label>
              Nome do serviço
              <input
                name="nome"
                autoFocus
                required
                maxLength={200}
                defaultValue={edicao.nome}
              />
            </label>
            <label>
              Duração (min)
              <input
                name="duracao"
                type="number"
                required
                min={1}
                max={1440}
                defaultValue={edicao.duracaoMinutos}
              />
            </label>
            <label>Preço (R$)<input name="preco" type="number" min="0" step="0.01" required defaultValue={(edicao.precoCentavos/100).toFixed(2)}/></label>
          </div>
          <div className="servico-campos">
            <label>Descrição do serviço<textarea name="descricao" maxLength={1000} rows={3} defaultValue={edicao.descricao} placeholder="Descreva o serviço para o cliente" /></label>
            <label>O que está incluído <small>Escreva um item por linha</small><textarea name="itens" rows={5} defaultValue={(edicao.itens || []).join("\n")} placeholder={"Lavagem da carroceria\nSecagem com microfibra"} /></label>
          </div>
          <fieldset className="servico-produtos"><legend>Produtos usados neste serviço</legend>{produtos === null ? <p>Carregando produtos…</p> : produtos.length === 0 ? <p>Cadastre produtos no estoque para selecioná-los aqui.</p> : produtos.map(produto => <label key={produto.id}><input type="checkbox" name="produtoIds" value={produto.id} defaultChecked={edicao.produtoIds?.includes(produto.id)} />{produto.nome}</label>)}</fieldset>
          <div className="acoes">
            <button className="primario" disabled={salvando || produtos === null}>
              {salvando ? "Salvando…" : "Salvar serviço"}
            </button>
            <button
              type="button"
              className="secundario"
              disabled={salvando}
              onClick={() => editar(null)}
            >
              Cancelar
            </button>
          </div>
        </form>
      )}
      {dados === null ? (
        <Carregamento erro={erro} tentar={tentar} />
      ) : (
        <>
          {erro && (
            <p className="erro" role="alert">
              {erro}
            </p>
          )}
          {aviso && (
            <p className="sucesso" role="status">
              {aviso}
            </p>
          )}
          <div className="quadro">
            <div className="quadro-topo">
              <h2>Catálogo de serviços</h2>
              <span className="contagem">{dados.length}</span>
            </div>
            {dados.length === 0 ? (
              <div className="vazio">
                <div className="icone-bloco">
                  <Icone nome="servicos" />
                </div>
                <h3>Nenhum serviço cadastrado</h3>
                <p>Adicione a primeira lavagem e sua duração.</p>
              </div>
            ) : (
              <ul className="catalogo">
                {dados.map((s) => (
                  <li key={s.id}>
                    <strong>{s.nome}</strong>
                    <span>
                      <Icone nome="clock" />
                      {s.duracaoMinutos} min · {(s.precoCentavos/100).toLocaleString("pt-BR",{style:"currency",currency:"BRL"})}
                    </span>
                    <div className="catalogo-acoes">
                    <button
                      className="texto"
                      disabled={salvando}
                      aria-label={"Editar " + s.nome}
                      onClick={() => editar(s)}
                    >
                      <Icone nome="pen" />
                      Editar
                    </button>
                    <button type="button" className="excluir-produto" disabled={salvando}
                      aria-label={"Excluir " + s.nome} onClick={() => setExclusaoPendente(s)}>Excluir</button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}
    </section>
  );
}

const dias = [
  "Segunda-feira",
  "Terça-feira",
  "Quarta-feira",
  "Quinta-feira",
  "Sexta-feira",
  "Sábado",
  "Domingo",
];
export function Funcionamento({ requisitar }: { requisitar: Requisitar }) {
  const [editando, setEditando] = useState(false);
  const { dados, definir, erro, setErro, tentar } = useDados<Config>(
    "configuracao",
    requisitar, editando,
  );
  const [salvando, setSalvando] = useState(false);
  const [aviso, setAviso] = useState("");
  function alterar(c: Config) {
    setEditando(true);
    definir(c);
    setErro("");
    setAviso("");
  }
  function periodo(index: number, valores: Partial<Periodo>) {
    if (dados)
      alterar({
        ...dados,
        horarios: dados.horarios.map((p, i) =>
          i === index ? { ...p, ...valores } : p,
        ),
      });
  }
  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!dados || salvando) return;
    setSalvando(true);
    setErro("");
    setAviso("");
    try {
      const c = await requisitar<Config>("configuracao", "PUT", dados);
      definir(c);
      setEditando(false);
      setAviso("Funcionamento salvo.");
    } catch (e) {
      setErro(mensagem(e));
    } finally {
      setSalvando(false);
    }
  }
  return (
    <section className="funcionamento">
      <div className="titulo">
        <div>
          <h1>Funcionamento</h1>
          <p>Controle a capacidade do lava-jato e os horários oferecidos.</p>
        </div>
      </div>
      {!dados ? (
        <Carregamento erro={erro} tentar={tentar} />
      ) : (
        <form onSubmit={salvar}>
          <fieldset disabled={salvando}>
            <div className="quadro editor funcionamento-capacidade">
              <h2>Capacidade da agenda</h2><label>Tolerância de chegada (minutos)<input type="number" min="0" max="1440" required value={dados.toleranciaMinutos} onChange={e=>alterar({...dados,toleranciaMinutos:Number(e.target.value)})}/></label><p>Defina quantos carros podem ser atendidos e como os horários serão oferecidos.</p>
              <div className="linha">
                <label>
                  Carros ao mesmo tempo
                  <input
                    type="number"
                    required
                    min={1}
                    max={2147483647}
                    value={dados.capacidade || ""}
                    placeholder="Ex.: 2"
                    onChange={(e) =>
                      alterar({ ...dados, capacidade: Number(e.target.value) })
                    }
                  />
                </label>
                <label>
                  Intervalo entre inícios (min)
                  <input
                    type="number"
                    required
                    min={1}
                    max={1440}
                    value={dados.intervaloMinutos || ""}
                    onChange={(e) =>
                      alterar({
                        ...dados,
                        intervaloMinutos: Number(e.target.value),
                      })
                    }
                  />
                </label>
              </div>
            </div>
            <div className="quadro funcionamento-semana">
              <div className="quadro-topo">
                <div><h2>Expediente semanal</h2><p>Adicione os períodos disponíveis em cada dia.</p></div>
              </div>
              {dias.map((dia, i) => (
                <div className="dia" key={dia}>
                  <div>
                    <strong>{dia}</strong>
                    <small className={dados.horarios.some((p) => p.diaSemana === i + 1) ? "dia-aberto" : "dia-fechado"}>{dados.horarios.some((p) => p.diaSemana === i + 1) ? "Aberto" : "Fechado"}</small>
                  </div>
                  <div className="periodos">
                    {dados.horarios.map(
                      (p, j) =>
                        p.diaSemana === i + 1 && (
                          <div className="periodo" key={j}>
                            <input
                              aria-label={"Abertura de " + dia}
                              required
                              type="time"
                              value={p.abre}
                              onChange={(e) =>
                                periodo(j, { abre: e.target.value })
                              }
                            />
                            <span>às</span>
                            <input
                              aria-label={"Fechamento de " + dia}
                              required
                              pattern="([01][0-9]|2[0-3]):[0-5][0-9]|24:00"
                              title="Horário em HH:MM, até 24:00"
                              value={p.fecha}
                              onChange={(e) =>
                                periodo(j, { fecha: e.target.value })
                              }
                            />
                            <button
                              type="button"
                              className="texto"
                              aria-label={"Remover período de " + dia}
                              onClick={() =>
                                alterar({
                                  ...dados,
                                  horarios: dados.horarios.filter(
                                    (_, k) => j !== k,
                                  ),
                                })
                              }
                            >
                              Remover
                            </button>
                          </div>
                        ),
                    )}
                    <button
                      type="button"
                      className="texto"
                      onClick={() =>
                        alterar({
                          ...dados,
                          horarios: [
                            ...dados.horarios,
                            { diaSemana: i + 1, abre: "08:00", fecha: "18:00" },
                          ],
                        })
                      }
                    >
                      <Icone nome="plus" />
                      Adicionar período
                    </button>
                  </div>
                </div>
              ))}
            </div>
            {erro && (
              <p className="erro" role="alert">
                {erro}
              </p>
            )}
            {aviso && (
              <p className="sucesso" role="status">
                {aviso}
              </p>
            )}
            <button className="primario salvar" disabled={salvando}>
              {salvando ? "Salvando…" : "Salvar funcionamento"}
            </button>
          </fieldset>
        </form>
      )}
    </section>
  );
}
