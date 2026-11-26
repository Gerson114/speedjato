"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { acompanhar } from "../../acompanhar";
import { mensagem, type Requisitar } from "../base";
type Produto={id:number;nome:string;unidade:string;quantidade:number;consumoDiario:number;diasRestantes:number};
export default function Estoque({requisitar}:{requisitar:Requisitar}) {
 const [lista,setLista]=useState<Produto[]>([]),[erro,setErro]=useState(""),[busy,setBusy]=useState(false),[cadastrando,setCadastrando]=useState(false);
 async function carregar(){try{setLista(await requisitar<Produto[]>("estoque"))}catch(e){setErro(mensagem(e))}}
 useEffect(()=>{
  return acompanhar(signal => requisitar<Produto[]>("estoque","GET",undefined,signal), setLista, e=>setErro(mensagem(e)));
 },[requisitar]);
 async function salvar(e:FormEvent<HTMLFormElement>){e.preventDefault();setBusy(true);setErro("");const f=new FormData(e.currentTarget);const unidade=String(f.get("unidade"));const fator=unidade==="L"?1000:1;const quantidade=Number(f.get("embalagens"))*Number(f.get("conteudo"))*fator;const consumoDiario=Number(f.get("consumo"))*(f.get("unidadeConsumo")==="L"?1000:1);if(!Number.isFinite(quantidade)||!Number.isFinite(consumoDiario)||quantidade<=0||consumoDiario<=0){setErro("Informe valores maiores que zero.");setBusy(false);return}try{await requisitar("estoque","POST",{nome:f.get("nome"),unidade:"ml",quantidade,consumoDiario});setCadastrando(false);await carregar()}catch(e){setErro(mensagem(e))}finally{setBusy(false)}}
 const [excluindo,setExcluindo]=useState<number|null>(null);
 const [produtoParaExcluir, setProdutoParaExcluir] = useState<Produto|null>(null);
 const dialogo = useRef<HTMLDialogElement>(null);
 useEffect(() => {
  if (produtoParaExcluir) dialogo.current?.showModal();
 }, [produtoParaExcluir]);
 async function excluir(p: Produto) {
  if (excluindo !== null) return;
  setExcluindo(p.id);
  setErro("");
  try {
   await requisitar("estoque/" + p.id, "DELETE");
   dialogo.current?.close();
   setLista(produtos => produtos.filter(produto => produto.id !== p.id));
  } catch (e) {
   dialogo.current?.close();
   setErro(mensagem(e));
  } finally {
   setExcluindo(null);
  }
 }
 async function consumir(p:Produto){const v=prompt("Quantidade consumida de "+p.nome+" ("+p.unidade+")");if(!v)return;try{await requisitar("estoque/"+p.id+"/consumo","POST",{quantidade:Number(v.replace(",","."))});await carregar()}catch(e){setErro(mensagem(e))}}
 return <section><div className="titulo"><div><h1>Estoque de produtos</h1><p>Acompanhe o consumo e estime quando será preciso repor.</p></div><button className="primario" onClick={()=>{setErro("");setCadastrando(true)}}>+ Cadastrar produto</button></div>{erro&&<p className="erro" role="alert">{erro}</p>}{cadastrando && <form className="quadro estoque-form" onSubmit={salvar}><h2>Cadastrar produto</h2><div><label>Produto<input name="nome" placeholder="Sabão líquido para carro" required/></label><label>Unidade comprada<select name="unidade"><option>ml</option><option>L</option></select></label><label>Embalagens compradas<input name="embalagens" type="number" min="1" step="1" placeholder="Ex. 5" required/></label><label>Conteúdo por embalagem<input name="conteudo" type="number" min="0.001" step="any" placeholder="Ex. 5" required/></label><label>Uso diário<input name="consumo" type="number" min="0.001" step="any" placeholder="Ex. 200" required/></label><label>Unidade do uso<select name="unidadeConsumo"><option>ml</option><option>L</option></select></label><div className="estoque-form-acoes"><button className="primario" disabled={busy}>{busy?"Salvando…":"Salvar produto"}</button><button type="button" className="secundario" disabled={busy} onClick={()=>{setCadastrando(false);setErro("")}}>Cancelar</button></div></div></form>}<div className="quadro estoque-lista"><div className="quadro-topo"><h2>Produtos cadastrados</h2><button className="texto direita" onClick={()=>void carregar()}>Atualizar</button></div>{!lista.length?<p className="vazio">Cadastre o primeiro produto para acompanhar o estoque.</p>:<div className="estoque-cartoes">{lista.map(p=><article className="estoque-cartao" key={p.id}><div className="estoque-cartao-topo"><span className="estoque-icone" aria-hidden="true">◈</span><div><small>PRODUTO</small><h3>{p.nome}</h3></div></div><div className="estoque-dados"><div><span>Quantidade restante</span><strong>{(p.quantidade/1000).toLocaleString("pt-BR")} L</strong><small>{p.quantidade.toLocaleString("pt-BR")} ml</small></div><div><span>Consumo diário</span><strong>{(p.consumoDiario/1000).toLocaleString("pt-BR")} L</strong><small>por dia</small></div></div><div className="estoque-cartao-rodape"><span>{p.consumoDiario?`Estimativa: ${Math.ceil(p.diasRestantes)} dias`:"Consumo não informado"}</span><button className="secundario" onClick={()=>void consumir(p)}>Registrar consumo</button><button className="excluir-produto" aria-label={`Excluir ${p.nome}`} disabled={excluindo !== null} onClick={()=>setProdutoParaExcluir(p)}>{excluindo === p.id ? "Excluindo…" : "Excluir produto"}</button></div></article>)}</div>}</div>
 <dialog ref={dialogo} className="confirmar-exclusao" aria-labelledby="titulo-exclusao"
  onClose={()=>setProdutoParaExcluir(null)} onCancel={e=>{if(excluindo !== null)e.preventDefault()}}>
  <h2 id="titulo-exclusao">Você quer excluir este produto?</h2>
  <p>O produto <strong>{produtoParaExcluir?.nome}</strong> será removido do estoque.</p>
  <div className="modal-acoes">
   <button autoFocus className="secundario" disabled={excluindo !== null} onClick={()=>dialogo.current?.close()}>Cancelar</button>
   <button className="excluir-produto" disabled={excluindo !== null} onClick={()=>{if(produtoParaExcluir)void excluir(produtoParaExcluir)}}>{excluindo !== null ? "Excluindo…" : "Sim, excluir"}</button>
  </div>
 </dialog></section>
}
