"use client";
import { useEffect, useState } from "react";
import Conversa from "./conversa";
import { mensagem, type Requisitar } from "../admin/base";

type Cliente = {id:number;nome:string;email:string;ultimaMensagem:string};
export default function ChatAdmin({requisitar}:{requisitar:Requisitar}){
 const [lista,setLista]=useState<Cliente[]>([]);
 const [selecionado,setSelecionado]=useState<Cliente|null>(null);
 const [erro,setErro]=useState("");
 const [carregando,setCarregando]=useState(true);
 useEffect(()=>{
  let ativo=true, consultando=false, pendente=false;
  let fonte:WebSocket|null=null;
  let reconectar:number;
  async function atualizar(){
   if(consultando){pendente=true;return}
   consultando=true;
   try{const dados=await requisitar<Cliente[]>("chat");if(ativo){setLista(dados);setErro("")}}
   catch(e){if(ativo)setErro(mensagem(e))}
   finally{consultando=false;if(ativo){setCarregando(false);if(pendente){pendente=false;void atualizar()}}}
  }
  function abrir(){
   if(document.hidden||fonte)return;
   fonte=new WebSocket(`${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/ws/chat/eventos`);
   fonte.onmessage=()=>void atualizar();
   fonte.onclose=()=>{fonte=null;if(ativo&&!document.hidden)reconectar=window.setTimeout(abrir,1500)};
  }
  function visibilidade(){if(document.hidden){window.clearTimeout(reconectar);fonte?.close()}else abrir()}
  abrir();document.addEventListener("visibilitychange",visibilidade);
  return()=>{ativo=false;window.clearTimeout(reconectar);fonte?.close();document.removeEventListener("visibilitychange",visibilidade)};
 },[requisitar]);
 return <section><div className="titulo"><div><h1>Conversas</h1><p>Mensagens dos clientes</p></div></div>
 {erro&&<p className="erro" role="alert">{erro}</p>}
 <div className="chat-layout"><aside className="chat-lista" aria-label="Clientes">
 {carregando?<p role="status">Carregando conversas…</p>:lista.length===0?<p>Nenhuma conversa iniciada.</p>:lista.map(c=><button key={c.id} type="button" className={selecionado?.id===c.id?"ativo":""} onClick={()=>setSelecionado(c)}><strong>{c.nome}</strong><span>{c.email}</span><small>{c.ultimaMensagem}</small></button>)}
 </aside><div className="chat-area">{selecionado?<><h2>{selecionado.nome}</h2><Conversa key={selecionado.id} rota={`chat/${selecionado.id}`} requisitar={requisitar} papel="admin"/></>:<p>Selecione uma conversa para responder.</p>}</div></div></section>
}
