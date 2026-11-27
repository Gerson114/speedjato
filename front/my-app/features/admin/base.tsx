import Image from "next/image";

export type Servico = { id: number; nome: string; duracaoMinutos: number; precoCentavos: number; descricao: string; itens: string[]; produtoIds: number[]; produtos: {id:number;nome:string}[] };
export type Reserva = {
  id: number;
  chegou: boolean;
  servico: string;
  cliente: string;
  inicio: string;
  fim: string;
  data: string;
  veiculo: string;
  precoCentavos: number;
  status: "pendente" | "confirmado" | "cancelado";
  acoes?: { confirmar: boolean; cancelar: boolean; etapas: Reserva["lavagemStatus"][] };
  lavagemStatus: "aguardando" | "proximo" | "em_andamento" | "concluida";
  email?: string;
  modelo?: string;
  placa?: string;
  cor?: string;
};
export type Periodo = { diaSemana: number; abre: string; fecha: string };
export type Config = {
  toleranciaMinutos: number;
  capacidade: number;
  intervaloMinutos: number;
  horarios: Periodo[];
};
export type Requisitar = <T>(
  path: string,
  method?: string,
  body?: unknown,
  signal?: AbortSignal,
) => Promise<T>;

export class Falha extends Error {
  constructor(
    public status: number,
    mensagem: string,
  ) {
    super(mensagem);
  }
}

export const api: Requisitar = async (path, method = "GET", body, signal) => {
  let resposta: Response;
  try {
    resposta = await fetch("/api/admin/" + path, {
      method,
      signal,
      cache: "no-store",
      headers: body ? { "Content-Type": "application/json" } : {},
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (erro) {
    if (erro instanceof DOMException && erro.name === "AbortError") throw erro;
    throw new Falha(0, "Não foi possível conectar ao servidor. Confira se o frontend e a API estão iniciados.");
  }
  const dados = await resposta.json().catch(() => ({}));
  if (!resposta.ok)
    throw new Falha(
      resposta.status,
      dados.error || (resposta.status === 503 ? "A API do lava-jato está indisponível. Verifique se o backend está iniciado." : "Não foi possível concluir a operação."),
    );
  return dados;
};

export function mensagem(e: unknown) {
  return e instanceof Falha ? e.message : "Falha inesperada. Tente novamente.";
}
export function Marca() {
  return (
    <span className="marca"><Image src="/speedjato.webp" width={594} height={314} alt="SpeedJato" className="logo-speedjato" /></span>
  );
}

const formas = {
  chat: <path d="M4 5h16v11H9l-5 4z" />,
  historico: <><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></>,
  visao: <path d="M4 3v17h17M8 16v-4m5 4V6m5 10V9" />,
  estoque: <path d="m3 7 9-4 9 4v10l-9 4-9-4zm0 0 9 4 9-4m-9 4v10M7 5l9 4" />,
  agenda: (
    <>
      <rect x="3" y="5" width="18" height="16" rx="3" />
      <path d="M7 3v4m10-4v4M3 11h18m-13 4h3" />
    </>
  ),
  servicos: (
    <>
      <path d="m3 15 2-6h13l3 6v4H3zm2 4v2m14-2v2M7 15h2m6 0h2" />
    </>
  ),
  config: (
    <>
      <path d="M4 7h7m5 0h4M4 17h3m5 0h8" />
      <circle cx="13" cy="7" r="2.5" />
      <circle cx="9" cy="17" r="2.5" />
    </>
  ),
  sair: (
    <>
      <path d="M10 4H5v16h5m3-4 4-4-4-4m-4 4h12" />
    </>
  ),
  lock: (
    <>
      <rect x="5" y="10" width="14" height="11" rx="2" />
      <path d="M8 10V7a4 4 0 0 1 8 0v3m-4 4v3" />
    </>
  ),
  eye: (
    <>
      <path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z" />
      <circle cx="12" cy="12" r="3" />
    </>
  ),
  hidden: (
    <>
      <path d="m3 3 18 18M3 9l-1 3s4 7 10 7c2 0 4-1 5-2M7 6c2-1 3-1 5-1 6 0 10 7 10 7l-2 3" />
    </>
  ),
  plus: <path d="M12 5v14M5 12h14" />,
  pen: (
    <>
      <path d="m4 17 12-12 3 3L7 20l-4 1zM14 7l3 3" />
    </>
  ),
  clock: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v6l4 2" />
    </>
  ),
};
export function Icone({ nome }: { nome: keyof typeof formas }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {formas[nome]}
    </svg>
  );
}
