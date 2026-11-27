"use client";
import { fecharModal } from "../fechar-modal";
import { useEffect, useState, type FormEvent } from "react";
import { acompanhar } from "../../acompanhar";
import { mensagem, type Requisitar, type Servico } from "../base";

export function AgendarBalcao({ requisitar, fechar, concluido, diaInicial }: { requisitar: Requisitar; fechar: () => void; diaInicial: string; concluido: (dia: string) => void }) {
  const [servicos,setServicos]=useState<Servico[]>([]);
  const [servicoId,setServicoId]=useState("");
  const [data,setData]=useState(diaInicial);
  const [horarios,setHorarios]=useState<string[]>([]);
  const [erro,setErro]=useState("");
  const [salvando,setSalvando]=useState(false);
  useEffect(() => acompanhar(signal => requisitar<Servico[]>("servicos","GET",undefined,signal), lista => {
    setServicos(lista);
    setServicoId(id => lista.some(s => String(s.id) === id) ? id : String(lista[0]?.id ?? ""));
  }, e => setErro(mensagem(e))), [requisitar]);
  useEffect(() => {
    if (!servicoId || !data) return;
    return acompanhar(signal => requisitar<string[]>("horarios?servicoId="+servicoId+"&data="+data,"GET",undefined,signal),
      setHorarios, e => setErro(mensagem(e)));
  }, [requisitar,servicoId,data]);
  async function salvar(e:FormEvent<HTMLFormElement>){e.preventDefault();setErro("");setSalvando(true);const f=new FormData(e.currentTarget);try{const reserva = await requisitar<{ data: string }>("agendamentos/balcao","POST",{nome:f.get("nome"),email:f.get("email"),modelo:f.get("modelo"),placa:f.get("placa"),cor:f.get("cor"),servicoId:Number(servicoId),data,inicio:f.get("inicio")});concluido(reserva.data)}catch(e){setErro(mensagem(e))}finally{setSalvando(false)}}
  return <div className="modal-fundo" onMouseDown={e=>{if(e.target===e.currentTarget)fecharModal(e.currentTarget, fechar)}}><section className="modal-detalhes" role="dialog" aria-modal="true" aria-labelledby="titulo-balcao"><div className="modal-topo"><div><p className="dash-etiqueta">NOVO ATENDIMENTO</p><h2 id="titulo-balcao">Agendar para um cliente</h2></div><button className="texto" onClick={e => fecharModal(e.currentTarget, fechar)}>Fechar</button></div><form className="form-balcao" onSubmit={salvar}><label>Nome do cliente<input name="nome" required maxLength={120}/></label><label>E-mail<input name="email" type="email" required maxLength={254}/></label><label>Veículo / modelo<input name="modelo" required maxLength={100} placeholder="Ex. Honda Civic"/></label><label>Placa<input name="placa" required maxLength={20}/></label><label>Cor<input name="cor" maxLength={40}/></label><label>Serviço<select value={servicoId} onChange={e=>setServicoId(e.target.value)} required>{servicos.map(s=><option key={s.id} value={s.id}>{s.nome}</option>)}</select></label><label>Data<input type="date" value={data} onChange={e=>setData(e.target.value)} required/></label><label>Horário disponível<select name="inicio" required disabled={!horarios.length}><option value="">{horarios.length?"Escolha um horário":"Sem horários disponíveis"}</option>{horarios.map(h=><option key={h}>{h}</option>)}</select></label>{erro&&<p className="erro" role="alert">{erro}</p>}<div className="modal-acoes"><button type="button" className="secundario" onClick={e => fecharModal(e.currentTarget, fechar)}>Cancelar</button><button className="primario" disabled={salvando||!horarios.length}>{salvando?"Agendando…":"Confirmar agendamento"}</button></div></form></section></div>
}
