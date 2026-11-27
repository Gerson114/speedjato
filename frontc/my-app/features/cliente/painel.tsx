"use client";
import { acompanhar } from "../acompanhar";
import Script from "next/script";
import Link from "next/link";
import{useEffect,useState,type FormEvent}from"react";
type Perfil={nome:string;email:string};type Servico={id:number;nome:string;duracaoMinutos:number;precoCentavos:number;descricao:string;itens:string[];produtos:{id:number;nome:string}[]};type Carro={id:number;apelido:string;modelo:string;placa:string;cor:string};type Reserva={id:number;servico:string;veiculo:string;data:string;inicio:string;fim:string;precoCentavos:number;modelo?:string;placa?:string;status:string;lavagemStatus:string;situacaoCliente:string};type Aviso={id:number;mensagem:string;criadoEm:string;lido:boolean};
class ErroDaApi extends Error {
 constructor(message: string, readonly status: number) { super(message); }
}
async function api<T>(path:string,method="GET",body?:unknown,signal?:AbortSignal):Promise<T>{const r=await fetch("/api/usuario/"+path,{method,signal,cache:"no-store",headers:body?{"Content-Type":"application/json",...(path==="google"?{"X-Requested-With":"XMLHttpRequest"}:{})}:{},body:body?JSON.stringify(body):undefined});const d=await r.json();if(!r.ok)throw new ErroDaApi(d.error||"Não foi possível continuar", r.status);return d}
const reais=(v:number)=>new Intl.NumberFormat("pt-BR",{style:"currency",currency:"BRL"}).format(v/100);
function Calendario({ hoje, valor, selecionar }: { hoje: string; valor: string; selecionar: (dia: string) => void }) {
  const [mes, setMes] = useState(() => (valor || hoje).slice(0, 7));
  const [ano, numeroMes] = mes.split("-").map(Number);
  const primeiroDia = new Date(Date.UTC(ano, numeroMes - 1, 1)).getUTCDay();
  const diasNoMes = new Date(Date.UTC(ano, numeroMes, 0)).getUTCDate();
  function mudarMes(delta: number) {
    const proximo = new Date(Date.UTC(ano, numeroMes - 1 + delta, 1));
    setMes(`${proximo.getUTCFullYear()}-${String(proximo.getUTCMonth() + 1).padStart(2, "0")}`);
  }
  const titulo = new Intl.DateTimeFormat("pt-BR", { month: "long", year: "numeric", timeZone: "UTC" }).format(new Date(Date.UTC(ano, numeroMes - 1, 1)));
  return <div className="calendario">
    <div className="calendario-topo"><button type="button" onClick={() => mudarMes(-1)} aria-label="Mês anterior">‹</button><strong>{titulo}</strong><button type="button" onClick={() => mudarMes(1)} aria-label="Próximo mês">›</button></div>
    <div className="calendario-grade">{["D", "S", "T", "Q", "Q", "S", "S"].map((dia, i) => <span key={i}>{dia}</span>)}
      {Array.from({ length: primeiroDia }, (_, i) => <span key={`vazio-${i}`} />)}
      {Array.from({ length: diasNoMes }, (_, i) => {
        const dia = `${mes}-${String(i + 1).padStart(2, "0")}`;
        return <button type="button" key={dia} disabled={dia < hoje} aria-pressed={dia === valor} aria-label={new Intl.DateTimeFormat("pt-BR", { dateStyle: "full", timeZone: "UTC" }).format(new Date(`${dia}T12:00:00Z`))} onClick={() => selecionar(dia)}>{i + 1}</button>;
      })}
    </div>
  </div>;
}
export default function PainelCliente({comCookie,googleId,nonce}:{comCookie:boolean;googleId?:string;nonce?:string}){
 const [seletorAberto, setSeletorAberto] = useState<"veiculo" | "data" | "hora" | null>(null);
 const [googlePronto,setGooglePronto]=useState(false);
 const [autenticando, setAutenticando] = useState(false);
 const [carregandoCatalogo, setCarregandoCatalogo] = useState(true);
 const[mostrarSenha,setMostrarSenha]=useState(false),[mostrarConfirmacao,setMostrarConfirmacao]=useState(false);
 const[secao,setSecao]=useState<"servicos"|"veiculos"|"agendamentos"|"avisos"|"conta">(comCookie ? "agendamentos" : "servicos"),[menuAberto,setMenuAberto]=useState(false),[etapaServico,setEtapaServico]=useState<"lista"|"detalhes"|"agendar">("lista");
 const[voltandoDaGaragem,setVoltandoDaGaragem]=useState(false);
 const[logado,setLogado]=useState(comCookie),[modo,setModo]=useState<"login"|"cadastro">("login"),[nome,setNome]=useState(""),[email,setEmail]=useState(""),[senha,setSenha]=useState(""),[confirmarSenha,setConfirmarSenha]=useState(""),[ocupado,setOcupado]=useState(false),[erro,setErro]=useState(""),[servicos,setServicos]=useState<Servico[]>([]),[carros,setCarros]=useState<Carro[]>([]),[reservas,setReservas]=useState<Reserva[]>([]),[veiculo,setVeiculo]=useState(""),[servico,setServico]=useState(""),[data,setData]=useState(""),[horarios,setHorarios]=useState<string[]>([]),[ocupados,setOcupados]=useState<string[]>([]),[hora,setHora]=useState(""),[carro,setCarro]=useState({modelo:"",placa:"",cor:""}),[mensagem,setMensagem]=useState("");
 const [paginaHistorico,setPaginaHistorico]=useState(1);
 const[avisos,setAvisos]=useState<Aviso[]>([]),[filtroHistorico,setFiltroHistorico]=useState("");
 useEffect(() => {
  if (secao !== "servicos" || autenticando) return;
  return acompanhar(signal => api<Servico[]>("servicos","GET",undefined,signal),
   lista => {
    setServicos(lista); setCarregandoCatalogo(false);
    if (servico && !lista.some(item => String(item.id) === servico)) {
     setServico(""); setHora(""); setEtapaServico("lista");
     setMensagem("Esse serviço foi retirado do catálogo.");
    }
   },
   e => { setErro(e.message); setCarregandoCatalogo(false); });
 }, [secao, autenticando, servico]);
 const [perfil,setPerfil]=useState<Perfil|null>(null);
 const [erroPerfil,setErroPerfil]=useState("");
 const [tentativaPerfil,setTentativaPerfil]=useState(0);
 const carregandoPerfil = !perfil && !erroPerfil;
 useEffect(() => {
  if (!logado || secao !== "conta") return;
  return acompanhar(signal => api<Perfil>("perfil","GET",undefined,signal),
   dados => { setPerfil(dados); setErroPerfil(""); }, e => setErroPerfil(e.message));
 }, [logado, secao, tentativaPerfil]);
 const [consultando,setConsultando]=useState(true);
 useEffect(() => {
  if (!logado || secao !== "agendamentos") return;
  const params = new URLSearchParams({pagina: String(paginaHistorico)});
  if (filtroHistorico) params.set("placa", filtroHistorico);
  return acompanhar(signal => api<Reserva[]>("agendamentos?"+params,"GET",undefined,signal),
   lista => { setReservas(lista); setConsultando(false); },
   e => { setErro(e.message); setConsultando(false); });
 }, [logado, secao, paginaHistorico, filtroHistorico]);
 useEffect(() => {
  if (!logado) return;
  function falha(e: Error) {
   if (e instanceof ErroDaApi && (e.status === 401 || e.status === 403)) {
    setLogado(false); setAutenticando(false); setSecao("servicos"); setEtapaServico("lista");
    setAvisos([]); setCarros([]); setReservas([]); setPerfil(null); setErro("");
   } else setErro(e.message);
  }
  const pararVeiculos = acompanhar(signal => api<Carro[]>("veiculos","GET",undefined,signal),
   lista => {
    setCarros(lista);
    setVeiculo(id => lista.some(c => String(c.id) === id) ? id : String(lista[0]?.id ?? ""));
   }, falha, 10000);
  const pararAvisos = acompanhar(signal => api<Aviso[]>("avisos","GET",undefined,signal),setAvisos,falha);
  return () => { pararVeiculos(); pararAvisos(); };
 }, [logado]);
 useEffect(() => {
  if (!logado || secao !== "servicos" || etapaServico !== "agendar" || !servico || !data) return;
  return acompanhar(signal => api<{horarios:string[];ocupados:string[]}>(
   `disponibilidade?servicoId=${servico}&data=${data}`,"GET",undefined,signal),
   ({horarios,ocupados}) => {
    setHorarios(horarios); setOcupados(ocupados);
    setHora(h => horarios.includes(h) && !ocupados.includes(h) ? h : "");
   }, e => setErro(e.message));
 }, [logado, secao, etapaServico, servico, data]);
 function concluirLogin() {
   setSenha("");
   setConfirmarSenha("");
   setMensagem("");
   setAutenticando(false);
   if (etapaServico !== "agendar") setSecao("agendamentos");
   setLogado(true);
 }
 function entrarGoogle() {
  if (!googleId || ocupado) return;
  setErro("");
  setOcupado(true);
  const google = (window as Window & {google?: {accounts: {oauth2: {
   initCodeClient(config: {client_id:string;scope:string;ux_mode:"popup";select_account:boolean;
    callback:(res:{code?:string})=>void;error_callback:(erro:{type:string})=>void}): {requestCode():void}
  }}}}).google;
  try {
   if (!google) throw Error("Google indisponível");
   google.accounts.oauth2.initCodeClient({
    client_id:googleId, scope:"openid email profile", ux_mode:"popup", select_account:true,
    callback:async ({code})=>{
     try {
      if (!code) throw Error("Login com Google cancelado.");
      await api("google","POST",{code});
      concluirLogin();
     } catch(e) {setErro((e as Error).message);}
     finally {setOcupado(false);}
    },
    error_callback:({type})=>{
     setOcupado(false);
     if (type === "popup_failed_to_open") setErro("Permita pop-ups para este site e clique novamente em Entrar com Google.");
     else if (type === "popup_closed") setErro("A janela do Google foi fechada antes de concluir o login. Tente novamente.");
     else setErro("Falha na comunicação com a janela do Google. Tente novamente ou entre com e-mail e senha.");
    }
   }).requestCode();
  } catch {setOcupado(false);setErro("Não foi possível abrir o Google.");}
 }
 async function enviar(e: FormEvent) {
  e.preventDefault();
  if (ocupado) return;
  setOcupado(true);
  setErro("");
  setMensagem("");
  const credenciais = {email: email.trim(), password: senha};
  try {
   if (modo === "cadastro") {
    if (senha !== confirmarSenha) throw Error("As senhas não coincidem.");
    await api("cadastro", "POST", {...credenciais, username: nome.trim()});
    setModo("login");
    setMensagem("Conta criada. Se necessário, entre para continuar.");
   }
   await api("login", "POST", credenciais);
   concluirLogin();
  } catch (e) {
   setErro((e as Error).message);
  } finally {
   setOcupado(false);
  }
 }
 async function cadastrarCarro(e:FormEvent){e.preventDefault();try{const c=await api<Carro>("veiculos","POST",{...carro,apelido:carro.modelo});setCarros([...carros,c]);setVeiculo(String(c.id));setCarro({modelo:"",placa:"",cor:""});setMensagem("Carro cadastrado.");if(voltandoDaGaragem){setVoltandoDaGaragem(false);setSecao("servicos")}}catch(e){setErro((e as Error).message)}}
 async function agendar(){setErro("");try{const reserva=await api<Reserva>("agendamentos","POST",{servicoId:Number(servico),veiculoId:Number(veiculo),data,inicio:hora});setReservas([reserva,...reservas]);setSecao("agendamentos");setMensagem("Horário reservado com sucesso.");}catch(e){setErro((e as Error).message)}}
 if (!logado && autenticando) return (
  <main className="entrada-cliente">
   <Link href="/" className="marca"><span className="simbolo" aria-hidden="true"><i/><i/><i/></span>SpeedJato</Link>
   <section className={`entrada-card ${modo === "cadastro" ? "modo-cadastro" : "modo-login"}`}>
    <div className="entrada-chamada">
     <p className="dash-etiqueta">CUIDADO PARA O SEU CARRO</p>
     <h1>{modo === "cadastro" ? <>Crie sua conta<span>.</span></> : <>Agende sua lavagem<span>.</span></>}</h1>
     <p>Escolha seu veículo, serviço e horário. Acompanhe cada etapa da lavagem por aqui.</p>
     <div className="entrada-pontos"><span>01 <b>Cadastre seu carro</b></span><span>02 <b>Escolha o serviço e horário</b></span><span>03 <b>Acompanhe seu agendamento</b></span></div>
    </div>
    <form className="entrada-form" onSubmit={enviar}>
     <button type="button" className="texto" disabled={ocupado} onClick={() => {setAutenticando(false); setEtapaServico("lista"); setErro(""); setMensagem("")}}>Voltar aos serviços</button>
     <p className="dash-etiqueta">{modo === "cadastro" ? "NOVO CLIENTE" : "ÁREA DO CLIENTE"}</p>
     <h2>{modo === "cadastro" ? "Criar conta" : "Bem-vindo de volta"}</h2>
     <p>{modo === "cadastro" ? "Leva poucos minutos para começar." : "Entre para continuar seu agendamento."}</p>
     {modo === "cadastro" && <label>Nome completo<input required maxLength={200} autoComplete="name" placeholder="Como podemos chamar você?" value={nome} onChange={e => setNome(e.target.value)} /></label>}
     <label>E-mail<input type="email" required maxLength={254} autoComplete="email" placeholder="voce@email.com" value={email} onChange={e => setEmail(e.target.value)} /></label>
     <label className="senha-label">Senha
      <span className="senha-controle"><input type={mostrarSenha ? "text" : "password"} required minLength={modo === "cadastro" ? 15 : 1} maxLength={72} autoComplete={modo === "cadastro" ? "new-password" : "current-password"} placeholder={modo === "cadastro" ? "Crie uma senha segura" : "Sua senha"} value={senha} onChange={e => setSenha(e.target.value)} /><button type="button" className="senha-mostrar" onClick={() => setMostrarSenha(v => !v)} aria-label={mostrarSenha ? "Ocultar senha" : "Mostrar senha"}>{mostrarSenha ? "Ocultar" : "Mostrar"}</button></span>
     </label>
     {modo === "cadastro" && <>
      <div className="senha-ajuda" aria-live="polite"><div className="senha-barra"><i style={{width: `${Math.min(senha.length / 15 * 100, 100)}%`}} /></div><span className={senha.length >= 15 ? "senha-valida" : ""}>{senha.length >= 15 ? "Tamanho mínimo atendido" : `${senha.length} de 15 caracteres mínimos`}</span><small>Uma frase fácil de lembrar funciona bem. Use pelo menos 15 caracteres.</small></div>
      <label className="senha-label">Confirme a senha
       <span className="senha-controle"><input type={mostrarConfirmacao ? "text" : "password"} required minLength={15} maxLength={72} autoComplete="new-password" placeholder="Digite a senha novamente" value={confirmarSenha} onChange={e => setConfirmarSenha(e.target.value)} /><button type="button" className="senha-mostrar" onClick={() => setMostrarConfirmacao(v => !v)} aria-label={mostrarConfirmacao ? "Ocultar confirmação" : "Mostrar confirmação"}>{mostrarConfirmacao ? "Ocultar" : "Mostrar"}</button></span>
       {confirmarSenha && <small className={confirmarSenha === senha ? "senha-valida" : "senha-invalida"}>{confirmarSenha === senha ? "As senhas coincidem" : "As senhas ainda não coincidem"}</small>}
      </label>
     </>}
     {erro && <p className="erro" role="alert">{erro}</p>}
     {mensagem && modo === "login" && <p className="sucesso" role="status">{mensagem}</p>}
     <button className="primario grande" disabled={ocupado}>{ocupado ? (modo === "cadastro" ? "Criando conta…" : "Entrando…") : modo === "cadastro" ? "Criar conta" : "Entrar"}</button>
     {googleId && <>
      <Script nonce={nonce} src="https://accounts.google.com/gsi/client" onReady={()=>setGooglePronto(true)} onError={()=>setErro("Não foi possível carregar o login com Google.")}/>
      <button type="button" className="secundario" disabled={ocupado || !googlePronto} onClick={entrarGoogle}>Entrar com o Google</button>
     </>}
     <p className="troca-modo">{modo === "cadastro" ? "Já tem uma conta?" : "Ainda não tem conta?"} <button type="button" className="texto" onClick={() => {setModo(modo === "cadastro" ? "login" : "cadastro"); setErro(""); setMostrarSenha(false); setMostrarConfirmacao(false)}}>{modo === "cadastro" ? "Fazer login" : "Criar cadastro"}</button></p>
    </form>
   </section>
   <p className="entrada-rodape">SpeedJato · Agendamento simples, cuidado de verdade.</p>
  </main>
 );
 const hoje=new Intl.DateTimeFormat("sv-SE",{timeZone:"America/Fortaleza"}).format(new Date());
 const trocarSecao=(aba:typeof secao)=>{setSecao(aba);setMenuAberto(false);setErro("");setMensagem("")};
 const servicoAtual=servicos.find(s=>String(s.id)===servico);
 const reservasVisiveis=reservas.slice(0,50);
 async function sair() {
  try {
   await api("sair", "POST");
   window.location.reload();
  } catch (e) {
   setErro((e as Error).message);
  }
 }
 return <main className="cliente-shell">
  {menuAberto&&<button className="menu-sombra" aria-label="Fechar menu" onClick={()=>setMenuAberto(false)}/>}
  <aside id="menu-cliente" className={`menu-cliente ${menuAberto?"menu-aberto":""}`}>
   <Link href="/" className="marca"><span className="simbolo" aria-hidden="true"><i/><i/><i/></span>SpeedJato</Link>
   <p className="menu-legenda">MENU PRINCIPAL</p>
   <nav className="menu-links" aria-label="Menu do cliente">
    <button className={secao==="servicos"?"ativo":""} aria-current={secao==="servicos"?"page":undefined} onClick={()=>trocarSecao("servicos")}><span aria-hidden="true">▦</span>Serviços</button>
    {logado && <><button className={secao==="veiculos"?"ativo":""} aria-current={secao==="veiculos"?"page":undefined} onClick={()=>trocarSecao("veiculos")}><span aria-hidden="true">▱</span>Meus veículos</button>
    <button className={secao==="agendamentos"?"ativo":""} aria-current={secao==="agendamentos"?"page":undefined} onClick={()=>trocarSecao("agendamentos")}><span aria-hidden="true">▤</span>Agendamentos</button>
    <button className={secao==="conta"?"ativo":""} aria-current={secao==="conta"?"page":undefined} onClick={()=>trocarSecao("conta")}><span aria-hidden="true">○</span>Minha conta</button>
    <button className={secao==="avisos"?"ativo":""} aria-current={secao==="avisos"?"page":undefined} onClick={()=>trocarSecao("avisos")}><span aria-hidden="true">◉</span>Avisos{avisos.length>0&&<small>{avisos.length}</small>}</button></>}
   </nav>
   {logado ? <div className="menu-rodape"><div className="perfil-cliente"><span className="perfil-selo">{email?email.charAt(0).toUpperCase():"C"}</span><span><strong>Área do cliente</strong><small>Seus agendamentos</small></span></div><button className="menu-sair" onClick={sair}><span aria-hidden="true">↪</span>Sair da conta</button></div> : <button className="menu-sair" onClick={() => {setModo("login");setAutenticando(true);setMenuAberto(false)}}>Entrar</button>}
  </aside>
  <div className="cliente-conteudo">
   <header className="cliente-topbar"><button className="menu-toggle" onClick={()=>setMenuAberto(true)} aria-label="Abrir menu" aria-controls="menu-cliente" aria-expanded={menuAberto}><span/><span/><span/></button><div><p className="dash-etiqueta">ÁREA DO CLIENTE</p><strong>{secao==="servicos"?"Agende sua lavagem":secao==="veiculos"?"Meus veículos":secao==="agendamentos"?"Meus agendamentos":secao==="conta"?"Minha conta":"Avisos e lembretes"}</strong></div><span className="topbar-marca"><span className="simbolo" aria-hidden="true"><i/><i/><i/></span>SpeedJato</span></header>
   {seletorAberto && <div className="seletor-fundo" onMouseDown={e => { if (e.target === e.currentTarget) setSeletorAberto(null); }} onKeyDown={e => { if (e.key === "Escape") setSeletorAberto(null); }}><section className="seletor-modal" role="dialog" aria-modal="true" aria-labelledby="seletor-titulo"><div className="seletor-topo"><h2 id="seletor-titulo">{seletorAberto === "veiculo" ? "Escolha seu veículo" : seletorAberto === "hora" ? "Escolha o horário" : "Escolha o dia"}</h2><button type="button" autoFocus onClick={() => setSeletorAberto(null)} aria-label="Fechar">×</button></div>{seletorAberto === "veiculo" ? <div className="seletor-veiculos">{carros.map(c => <button type="button" key={c.id} aria-pressed={String(c.id) === veiculo} onClick={() => { setVeiculo(String(c.id)); setSeletorAberto(null); }}><strong>{c.modelo}</strong><span>{c.placa}{c.cor ? ` · ${c.cor}` : ""}</span></button>)}</div> : seletorAberto === "hora" ? <div className="seletor-horarios"><p>Selecione um horário disponível para sua lavagem.</p><div className="grade-horarios">{horarios.map(h => <button type="button" key={h} disabled={ocupados.includes(h)} aria-pressed={hora === h} onClick={() => { setHora(h); setSeletorAberto(null); }}><strong>{h}</strong><span>{ocupados.includes(h) ? "Ocupado" : "Disponível"}</span></button>)}</div></div> : <Calendario hoje={hoje} valor={data} selecionar={dia => { setData(dia); setHora(""); setHorarios([]); setOcupados([]); setSeletorAberto(null); }} />}</section></div>}
   <div className="cliente-pagina">
    {erro&&<p role="alert" className="erro">{erro}</p>}{mensagem&&<p role="status" className="sucesso">{mensagem}</p>}
    {secao==="servicos"&&<>
     {etapaServico==="lista"&&<>
      <section className="pagina-intro"><p className="dash-etiqueta">ESCOLHA SEU CUIDADO</p><h1>Serviços de lavagem</h1><p>Toque em um serviço para ver preço, descrição e produtos incluídos.</p></section>
      {carregandoCatalogo?<p role="status">Carregando serviços…</p>:servicos.length===0?<div className="servicos-vazio">Nenhum serviço disponível no momento.</div>:<div className="servicos-cards">{servicos.map(s=>{return <button type="button" key={s.id} className="servico-card" onClick={()=>{setServico(String(s.id));setHora("");setEtapaServico("detalhes")}}><span className="servico-card-topo"><span className="servico-tag">SPEEDJATO</span><span className="servico-duracao">◷ {s.duracaoMinutos} min</span></span><strong className="servico-nome">{s.nome}</strong><span className="servico-resumo">{s.descricao}</span><span className="servico-preco">{reais(s.precoCentavos)}</span><span className="servico-abrir">Ver detalhes <b aria-hidden="true">→</b></span></button>})}</div>}
     </>}
     {etapaServico==="detalhes"&&servicoAtual&&(()=>{return <section className="servico-detalhe"><button className="voltar-servicos" onClick={()=>setEtapaServico("lista")}>← Voltar aos serviços</button><div className="detalhe-cabecalho"><p className="dash-etiqueta">DETALHES DO SERVIÇO</p><h1>{servicoAtual.nome}</h1><p>{servicoAtual.descricao}</p></div><div className="detalhe-resumo"><div><span>PREÇO</span><strong>{reais(servicoAtual.precoCentavos)}</strong></div><div><span>DURAÇÃO ESTIMADA</span><strong>{servicoAtual.duracaoMinutos} minutos</strong></div></div>{servicoAtual.itens?.length > 0 && <section className="detalhe-inclui"><p className="dash-etiqueta">CUIDADO INCLUÍDO</p><h2>O que fazemos nesta lavagem</h2><ul>{servicoAtual.itens.map(item=><li key={item}><span aria-hidden="true">✓</span>{item}</li>)}</ul></section>}{servicoAtual.produtos?.length > 0 && <section className="detalhe-inclui"><h2>Produtos utilizados</h2><ul>{servicoAtual.produtos.map(produto=><li key={produto.id}><span aria-hidden="true">✓</span>{produto.nome}</li>)}</ul></section>}<div className="detalhe-cta"><div><strong>Gostou deste serviço?</strong><span>Escolha seu carro e encontre um horário.</span></div><button className="primario" onClick={()=>{setEtapaServico("agendar");
      if (!logado) {
       setModo("cadastro");
       setAutenticando(true);
       setErro("");
       setMensagem("");
       return;
      }
      if(carros.length===0){setVoltandoDaGaragem(true);trocarSecao("veiculos")}}}>Quero agendar</button></div></section>})()}
     {etapaServico==="agendar"&&servicoAtual&&<section className="agendar-servico"><button className="voltar-servicos" onClick={()=>setEtapaServico("detalhes")}>← Voltar aos detalhes</button><div className="pagina-intro"><p className="dash-etiqueta">RESERVAR HORÁRIO</p><h1>{servicoAtual.nome}</h1><p>{reais(servicoAtual.precoCentavos)} · {servicoAtual.duracaoMinutos} minutos</p></div>{carros.length===0?<div className="sem-carro"><div><strong>Cadastre um carro para continuar</strong><p>O modelo e a placa são necessários para concluir a reserva.</p></div><button className="primario" onClick={()=>{setVoltandoDaGaragem(true);trocarSecao("veiculos")}}>Ir para meus veículos</button></div>:<div className="agendar-form"><div className="selecao-agendamento"><span>Veículo</span><button type="button" onClick={()=>setSeletorAberto("veiculo")}>{carros.find(c=>String(c.id)===veiculo)?.modelo || "Escolher veículo"} {carros.find(c=>String(c.id)===veiculo)?.placa && `· ${carros.find(c=>String(c.id)===veiculo)?.placa}`}</button></div><div className="selecao-agendamento"><span>Dia da lavagem</span><button type="button" onClick={()=>setSeletorAberto("data")}>{data ? data.split("-").reverse().join("/") : "Escolher uma data"}</button></div><div className="horarios-bloco"><h3>Escolha o horário</h3>{!data?<p>Escolha uma data para consultar os horários.</p>:horarios.length===0?<p>Não há horários neste dia.</p>:<><p>Selecione um dos horários disponíveis. Horários ocupados não podem ser escolhidos.</p><div className="selecao-agendamento"><button type="button" onClick={() => setSeletorAberto("hora")}>{hora || "Escolher horário"}</button></div></>}</div><button className="primario" disabled={!veiculo||!data||!hora} onClick={agendar}>Confirmar agendamento</button></div>}</section>}
    </>}
    {secao==="veiculos"&&<>
     <section className="pagina-intro"><p className="dash-etiqueta">GARAGEM</p><h1>Meus veículos</h1><p>Cadastre seus carros para agilizar os próximos agendamentos.</p></section>
     <section className="veiculos-layout"><form className="quadro cadastro-veiculo" onSubmit={cadastrarCarro}><p className="dash-etiqueta">NOVO VEÍCULO</p><h2>Adicionar carro</h2><label>Modelo do carro<input required maxLength={100} placeholder="Ex.: Honda Civic" value={carro.modelo} onChange={e=>setCarro({...carro,modelo:e.target.value})}/></label><label>Placa<input required maxLength={12} placeholder="ABC1D23" value={carro.placa} onChange={e=>setCarro({...carro,placa:e.target.value.toUpperCase()})}/></label><label>Cor <span className="opcional">Opcional</span><input maxLength={60} placeholder="Ex.: prata" value={carro.cor} onChange={e=>setCarro({...carro,cor:e.target.value})}/></label><button className="primario" type="submit">Salvar veículo</button></form>
      <div className="lista-veiculos"><div className="lista-cabecalho"><div><p className="dash-etiqueta">SUA GARAGEM</p><h2>Veículos cadastrados</h2></div><span className="contagem-veiculos">{carros.length}</span></div>{carros.length===0?<div className="estado-vazio">Nenhum veículo cadastrado. Adicione seu primeiro carro ao lado.</div>:carros.map(c=><article className="veiculo-item" key={c.id}><span className="carro-icone" aria-hidden="true">▱</span><div><strong>{c.modelo}</strong><span>{c.placa}{c.cor?` · ${c.cor}`:""}</span></div><button className={`veiculo-selecao ${veiculo===String(c.id)?"atual":""}`} onClick={()=>{setVeiculo(String(c.id));setMensagem(`${c.modelo} selecionado para o próximo agendamento.`)}}>{veiculo===String(c.id)?"Selecionado":"Usar no agendamento"}</button></article>)}</div>
     </section>
    </>}
    {secao==="agendamentos"&&<>
     <section className="pagina-intro"><p className="dash-etiqueta">MEUS AGENDAMENTOS</p><h1>Andamento da lavagem</h1><p>Veja a situação de cada carro. Esta tela se atualiza automaticamente.</p></section>
     <label className="filtro-historico">Filtrar veículo<select value={filtroHistorico} onChange={e=>{setPaginaHistorico(1);setFiltroHistorico(e.target.value)}}><option value="">Todos os veículos</option>{carros.map(c=><option key={c.id} value={c.placa}>{c.modelo} · {c.placa}</option>)}</select></label>
     <section className="historico-lista">{consultando?<p role="status">Consultando suas lavagens…</p>:reservasVisiveis.length===0?<div className="estado-vazio">Nenhum agendamento encontrado. <button className="texto" onClick={()=>trocarSecao("servicos")}>Ver serviços</button></div>:reservasVisiveis.map(r=><article key={r.id}><div className="historico-data"><strong>{r.data}</strong><span>{r.inicio}–{r.fim}</span></div><div className="historico-info"><strong>{r.modelo||r.veiculo}{r.placa?` · ${r.placa}`:""}</strong><span>{r.servico}</span></div><strong className="historico-preco">{reais(r.precoCentavos)}</strong><span className="cliente-status" data-etapa={r.status==="cancelado"?"cancelado":r.lavagemStatus}>{r.situacaoCliente}</span></article>)}</section><div className="paginacao-historico"><button className="secundario" disabled={paginaHistorico===1} onClick={()=>setPaginaHistorico(p=>p-1)}>Anterior</button><span>Página {paginaHistorico}</span><button className="secundario" disabled={reservas.length<=50} onClick={()=>setPaginaHistorico(p=>p+1)}>Próxima</button></div>
    </>}
    {secao==="conta"&&<>
     <section className="pagina-intro"><p className="dash-etiqueta">PERFIL</p><h1>Minha conta</h1><p>Confira os dados cadastrados e acesse seus veículos.</p></section>
     <section className="conta-painel">
      <div className="conta-cabecalho"><span className="conta-inicial" aria-hidden="true">{perfil?.nome?.charAt(0).toUpperCase() || "C"}</span><div><h2>Dados pessoais</h2><p>Informações da sua conta</p></div></div>
      {carregandoPerfil ? <p role="status">Carregando seus dados…</p> : erroPerfil ? <div><p className="erro" role="alert">{erroPerfil}</p><button className="secundario" onClick={()=>setTentativaPerfil(n=>n+1)}>Tentar novamente</button></div> : perfil && <dl className="conta-dados"><div><dt>Nome</dt><dd>{perfil.nome}</dd></div><div><dt>E-mail</dt><dd>{perfil.email}</dd></div></dl>}
      <div className="conta-atalhos"><div><strong>Meus veículos</strong><p>{carros.length === 1 ? "1 veículo cadastrado" : `${carros.length} veículos cadastrados`}</p></div><button className="secundario" onClick={()=>trocarSecao("veiculos")}>Ver veículos</button></div>
     </section>
    </>}
    {secao==="avisos"&&<>
     <section className="pagina-intro"><p className="dash-etiqueta">ATUALIZAÇÕES</p><h1>Avisos e lembretes</h1><p>Confirmações, lembretes de horário e andamento da lavagem.</p></section>
     <section className="avisos-lista">{avisos.length===0?<div className="estado-vazio">Você verá aqui as atualizações dos seus agendamentos.</div>:avisos.map(a=><article key={a.id}><span className="aviso-sinal" aria-hidden="true">•</span><div><strong>{a.mensagem}</strong><time>{a.criadoEm}</time></div></article>)}</section>
    </>}
   </div>
   <footer className="cliente-footer"><div><strong>SpeedJato<span>.</span></strong><p>Cuidado para o seu carro, no seu tempo.</p></div><nav aria-label="Navegação do rodapé"><button onClick={()=>trocarSecao("servicos")}>Serviços</button><button onClick={()=>trocarSecao("veiculos")}>Meus veículos</button><button onClick={()=>trocarSecao("agendamentos")}>Agendamentos</button></nav><small>© {new Date().getFullYear()} SpeedJato · Área do cliente</small></footer>
  </div>
 </main>;
}
