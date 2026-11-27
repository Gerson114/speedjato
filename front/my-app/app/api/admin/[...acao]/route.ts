import { isIP } from "node:net";
import { NextRequest, NextResponse } from "next/server";

type Context = { params: Promise<{ acao: string[] }> };

async function encaminhar(req: NextRequest, context: Context) {
  const { acao } = await context.params;
  const path = acao.join("/");
  const method = req.method;
  const origin = req.headers.get("origin");
  if (
    method !== "GET" &&
    ((origin && origin !== (process.env.APP_ORIGIN || req.nextUrl.origin)) ||
      req.headers.get("sec-fetch-site") === "cross-site")
  ) {
    return NextResponse.json(
      { error: "Origem não permitida" },
      { status: 403 },
    );
  }
  const login = method === "POST" && path === "login";
  const permitido =
    login ||
    (path === "sair" && method === "POST") ||
    (path === "servicos" && ["GET", "POST"].includes(method)) ||
    (/^estoque\/[1-9]\d*$/.test(path) && method === "DELETE") ||
    (path === "estoque" && ["GET", "POST"].includes(method)) ||
    (/^estoque\/[1-9]\d*\/consumo$/.test(path) && method === "POST") ||
    (/^servicos\/[1-9]\d*$/.test(path) && ["PUT", "DELETE"].includes(method)) ||
    (path === "configuracao" && ["GET", "PUT"].includes(method)) ||
    (["agendamentos", "agendamentos/novidades"].includes(path) && method === "GET") ||
    (path === "agendamentos/balcao" && method === "POST") ||
    (/^agendamentos\/[1-9]\d*\/chegada$/.test(path) && method === "PUT") ||
    (path === "horarios" && method === "GET") ||
    (/^agendamentos\/[1-9]\d*\/status$/.test(path) && method === "PUT") ||
    (/^agendamentos\/[1-9]\d*\/lavagem$/.test(path) && method === "PUT") ||
    (path === "veiculos" && method === "GET") ||
    (path === "resumo" && method === "GET") ||
    (path === "relatorios" && method === "GET");
  if (!permitido)
    return NextResponse.json({ error: "Rota inválida" }, { status: 404 });
  const sessao = req.cookies.get("admin_token");
  if (!login && !sessao)
    return NextResponse.json(
      { error: "Entre para acessar o painel" },
      { status: 401 },
    );

  const endpoint = login
    ? "/auth/login?painel=admin"
    : path === "configuracao"
      ? "/admin/agenda/configuracao"
      : "/admin/" + path;
  const url = new URL(
    endpoint,
    process.env.BACKEND_URL || "http://127.0.0.1:8081",
  );
  if (path === "agendamentos/novidades")
    url.searchParams.set("versao", req.nextUrl.searchParams.get("versao") || "");
  if (path === "agendamentos")
    url.searchParams.set("data", req.nextUrl.searchParams.get("data") || "");
  if ((path === "resumo" || path === "relatorios") && req.nextUrl.searchParams.has("periodo"))
    url.searchParams.set("periodo", req.nextUrl.searchParams.get("periodo")!);
  if (path === "resumo") {
    url.searchParams.set("de", req.nextUrl.searchParams.get("de") || "");
    url.searchParams.set("ate", req.nextUrl.searchParams.get("ate") || "");
  }
  if (path === "relatorios") {
    url.searchParams.set("de", req.nextUrl.searchParams.get("de") || "");
    url.searchParams.set("ate", req.nextUrl.searchParams.get("ate") || "");
  }
  if (path === "horarios") {
    for (const key of ["servicoId", "data"]) url.searchParams.set(key, req.nextUrl.searchParams.get(key) || "");
  }
  if (path === "agendamentos") {
    for (const key of ["cliente", "placa", "servico", "status", "lavagem", "visao", "categoria", "busca", "pagina"])
      if (req.nextUrl.searchParams.has(key)) url.searchParams.set(key, req.nextUrl.searchParams.get(key)!);
  }
  const headers = new Headers();
  if (!login && sessao)
    headers.set("Cookie", "admin_token=" + encodeURIComponent(sessao.value));
  if (origin) headers.set("Origin", origin);
  if (process.env.GATEWAY_TOKEN) {
    headers.set("X-Gateway-Token", process.env.GATEWAY_TOKEN);
    const nome = process.env.TRUSTED_CLIENT_IP_HEADER;
    const bruto = nome ? req.headers.get(nome) : null;
    const ip = (nome?.toLowerCase() === "x-forwarded-for" ? bruto?.split(",").at(-1) : bruto)?.trim();
    if (ip && isIP(ip)) headers.set("X-Client-IP", ip);
  }
  if (method !== "GET" && method !== "DELETE" && path !== "sair") {
    if (req.headers.get("content-type")?.split(";")[0] !== "application/json") {
      return NextResponse.json({ error: "Envie JSON" }, { status: 415 });
    }
    headers.set("content-type", "application/json");
  }
  try {
    const init: RequestInit & { duplex?: "half" } = {
      method,
      headers,
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.any([req.signal, AbortSignal.timeout(20000)]),
    };
    if (method !== "GET" && method !== "DELETE" && path !== "sair") {
      init.body = req.body;
      init.duplex = "half";
    }
    const upstream = await fetch(url, init);
    const response = new NextResponse(upstream.body, {
      status: upstream.status,
      headers: {
        "Content-Type":
          upstream.headers.get("Content-Type") || "application/json",
        "Cache-Control": "no-store",
      },
    });
    if (path === "sair" && upstream.ok) response.cookies.delete("admin_token");
    if (login) {
      for (const cookie of upstream.headers.getSetCookie()) {
        if (cookie.startsWith("admin_token="))
          response.headers.append("Set-Cookie", cookie);
      }
    } else if (upstream.status === 401) response.cookies.delete("admin_token");
    return response;
  } catch {
    return NextResponse.json(
      { error: "A API do lava-jato está indisponível. Tente novamente." },
      { status: 503 },
    );
  }
}
export { encaminhar as GET, encaminhar as POST, encaminhar as PUT, encaminhar as DELETE };
