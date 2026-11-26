import { Suspense } from "react";
import { cookies } from "next/headers";
import Painel from "../../features/admin/painel";

async function Entrada() {
  const sessao = (await cookies()).has("admin_token");
  return <Painel comCookie={sessao} />;
}
export default function Admin() {
  return (
    <Suspense
      fallback={
        <main className="aguarde" role="status">
          Abrindo o painel…
        </main>
      }
    >
      <Entrada />
    </Suspense>
  );
}
