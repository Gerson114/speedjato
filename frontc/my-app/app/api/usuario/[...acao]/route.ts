import { isIP } from "node:net";
import { NextRequest, NextResponse } from "next/server";
type Context={params:Promise<{acao:string[]}>};
async function encaminhar(req:NextRequest,{params}:Context){
 const path=(await params).acao.join("/"),method=req.method;
 const login=(path==="login"||path==="google")&&method==="POST",cadastro=path==="cadastro"&&method==="POST";
 const permitido=login||cadastro||(path==="sair"&&method==="POST")||(path==="servicos"&&method==="GET")||(path==="veiculos"&&["GET","POST"].includes(method))||(path==="agendamentos"&&["GET","POST"].includes(method))||(path==="avisos"&&method==="GET")||(path==="perfil"&&method==="GET")||(path==="horarios"&&method==="GET")||(path==="disponibilidade"&&method==="GET");
 if(!permitido)return NextResponse.json({error:"Rota inválida"},{status:404});
  const origin = req.headers.get("origin");
  if (path === "google" && (!origin || req.headers.get("x-requested-with") !== "XMLHttpRequest")) return NextResponse.json({error:"Origem não permitida"},{status:403});
  const origemDeTeste = process.env.NODE_ENV === "development" &&
    process.env.LAVA_REDE_IPS?.split(",").some(ip =>
      origin === `http://${ip}:${process.env.LAVA_REDE_PORTA}` &&
      req.headers.get("host") === `${ip}:${process.env.LAVA_REDE_PORTA}`
    );
  if (
    method !== "GET" &&
    ((origin && !origemDeTeste && origin !== (process.env.APP_ORIGIN || req.nextUrl.origin)) ||
      req.headers.get("sec-fetch-site") === "cross-site")
  ) {
    return NextResponse.json(
      { error: "Origem não permitida" },
      { status: 403 },
    );
  }
 const cookie=req.cookies.get("token");if(!login&&!cadastro&&path!=="servicos"&&!cookie)return NextResponse.json({error:"Entre para continuar"},{status:401});
 const endpoint=login?"/auth/"+path:cadastro?"/auth/cadastrar":"/usuario/"+path;const url=new URL(endpoint,process.env.BACKEND_URL||"http://127.0.0.1:8081");
 for(const key of ["data","servicoId","veiculoId","pagina","placa","antes"])if(req.nextUrl.searchParams.has(key))url.searchParams.set(key,req.nextUrl.searchParams.get(key)!);
 const headers=new Headers();if(cookie)headers.set("Cookie","token="+encodeURIComponent(cookie.value));if(process.env.GATEWAY_TOKEN)headers.set("X-Gateway-Token",process.env.GATEWAY_TOKEN);
 if (process.env.GATEWAY_TOKEN) {
   const nome = process.env.TRUSTED_CLIENT_IP_HEADER;
   const bruto = nome ? req.headers.get(nome) : null;
   const ip = (nome?.toLowerCase() === "x-forwarded-for" ? bruto?.split(",").at(-1) : bruto)?.trim();
   if (ip && isIP(ip)) headers.set("X-Client-IP", ip);
 }
 if(method!=="GET" && path!=="sair"){if(req.headers.get("content-type")?.split(";")[0]!=="application/json")return NextResponse.json({error:"Envie JSON"},{status:415});headers.set("Content-Type","application/json")}
 try{const init:RequestInit&{duplex?:"half"}={method,headers,redirect:"error",cache:"no-store",signal:AbortSignal.any([req.signal, AbortSignal.timeout(20000)])};if(method!=="GET" && path!=="sair"){init.body=req.body;init.duplex="half"}const upstream=await fetch(url,init);const response=new NextResponse(upstream.body,{status:upstream.status,headers:{"Content-Type":upstream.headers.get("Content-Type")||"application/json","Cache-Control":"no-store"}});if(path==="sair" && upstream.ok)response.cookies.delete("token");if(login)for(const c of upstream.headers.getSetCookie())if(c.startsWith("token="))response.headers.append("Set-Cookie",c);return response}catch{return NextResponse.json({error:"Não foi possível conectar à agenda."},{status:503})}
}
export{encaminhar as GET,encaminhar as POST};
