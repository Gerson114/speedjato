"use client";
import { useEffect, useState, type FormEvent } from "react";

type Mensagem = {id:number; autor:"admin"|"user"; texto:string; criadoEm:string};
type Requisitar = <T>(path:string, method?:string, body?:unknown) => Promise<T>;

export default function Conversa({rota, requisitar, papel}:{rota:string; requisitar:Requisitar; papel:"admin"|"user"}) {
 const [mensagens,setMensagens]=useState<Mensagem[]>([]);
 const [texto,setTexto]=useState("");
 const [erro,setErro]=useState("");
 const [carregando,setCarregando]=useState(true);
 const [enviando,setEnviando]=useState(false);
 const [temAnteriores,setTemAnteriores]=useState(false);
 useEffect(()=>{
  let ativo=true;
  let consultando=false;
  let pendente=false;
  let fonte:WebSocket|null=null;
  let reconectar:number;
  async function atualizar() {
   if(consultando){pendente=true;return}
   consultando=true;
   try {
    const lista=await requisitar<Mensagem[]>(rota);
    if(ativo){setMensagens(atual=>{
     const anteriores=atual.filter(m=>!lista.some(n=>n.id===m.id));
     return [...anteriores,...lista].sort((a,b)=>a.id-b.id);
    });setTemAnteriores(atual=>atual||lista.length===50);setErro("");}
   } catch(e){if(ativo)setErro((e as Error).message);}
   finally{consultando=false;if(ativo){setCarregando(false);if(pendente){pendente=false;void atualizar()}}}
  }
  function abrir(){
   if(document.hidden||fonte)return;
   fonte=new WebSocket(`${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/ws/${rota}/eventos`);
   fonte.onmessage=()=>void atualizar();
   fonte.onclose=()=>{fonte=null;if(ativo&&!document.hidden)reconectar=window.setTimeout(abrir,1500)};
  }
  function visibilidade(){if(document.hidden){window.clearTimeout(reconectar);fonte?.close()}else abrir()}
  abrir();
  document.addEventListener("visibilitychange",visibilidade);
  return()=>{ativo=false;window.clearTimeout(reconectar);fonte?.close();document.removeEventListener("visibilitychange",visibilidade)};
 },[rota,requisitar]);
 async function anteriores(){
  if(!mensagens.length)return;
  try{
   const lista=await requisitar<Mensagem[]>(rota+"?antes="+mensagens[0].id);
   setMensagens(atual=>[...lista.filter(m=>!atual.some(n=>n.id===m.id)),...atual]);
   setTemAnteriores(lista.length===50);
  }catch(e){setErro((e as Error).message)}
 }
 async function enviar(e:FormEvent){
  e.preventDefault();
  if(enviando||!texto.trim())return;
  setEnviando(true);setErro("");
  try{
   const m=await requisitar<Mensagem>(rota,"POST",{texto});
   setMensagens(atual=>atual.some(item=>item.id===m.id)?atual:[...atual,m].sort((a,b)=>a.id-b.id));
   setTexto("");
  }catch(e){setErro((e as Error).message)}
  finally{setEnviando(false)}
 }
 return <div className="chat-conversa">
  <div className="chat-historico" aria-label="Mensagens">
   {temAnteriores&&<button type="button" className="texto" onClick={()=>void anteriores()}>Carregar mensagens anteriores</button>}
   {carregando?<p role="status">Carregando conversa…</p>:mensagens.length===0?<p>Envie uma mensagem para iniciar a conversa.</p>:mensagens.map(m=><article key={m.id} className={m.autor===papel?"chat-mensagem minha":"chat-mensagem"}><strong>{m.autor==="admin"?"Lava-jato":"Cliente"}</strong><p>{m.texto}</p><time dateTime={m.criadoEm}>{new Date(m.criadoEm).toLocaleString("pt-BR",{dateStyle:"short",timeStyle:"short"})}</time></article>)}
  </div>
  {erro&&<p className="erro" role="alert">{erro}</p>}
  <form className="chat-envio" onSubmit={enviar}><label htmlFor="chat-texto">Mensagem</label><textarea id="chat-texto" maxLength={2000} value={texto} onChange={e=>setTexto(e.target.value)} placeholder="Escreva sua mensagem" rows={2}/><button className="primario" disabled={enviando||!texto.trim()}>{enviando?"Enviando…":"Enviar"}</button></form>
 </div>;
}
