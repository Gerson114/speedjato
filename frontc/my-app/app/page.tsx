import { Suspense } from "react";
import { cookies, headers } from "next/headers";
import PainelCliente from "../features/cliente/painel";

export default function Home() {
  return (
    <Suspense fallback={<main className="aguarde">Carregando…</main>}>
      <Entrada />
    </Suspense>
  );
}

async function Entrada() {
  const sessao = (await cookies()).has("token");
  return <PainelCliente nonce={(await headers()).get("x-nonce") ?? undefined} comCookie={sessao} googleId={process.env.GOOGLE_CLIENT_ID} />;
}
