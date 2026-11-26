import type { Reserva } from "../base";

export function deslocarDia(data: string, dias: number) {
  const dia = new Date(data + "T12:00:00Z");
  dia.setUTCDate(dia.getUTCDate() + dias);
  return dia.toISOString().slice(0, 10);
}

function campoCsv(valor: string) {
  const texto = /^[\s]*[=+@-]|^[\t\r\n]/.test(valor) ? "'" + valor : valor;
  return '"' + texto.replaceAll('"', '""') + '"';
}

export const etapas = { aguardando: "Aguardando", proximo: "Próximo na fila", em_andamento: "Em lavagem", concluida: "Concluída" };

export function montarCsv(reservas: Reserva[]) {
  const cabecalho = ["Data", "Início", "Fim", "Cliente", "Placa", "Serviço", "Reserva", "Lavagem", "Valor (R$)"];

  const linhas = reservas.map(r => [
    r.data.split("-").reverse().join("/"), r.inicio, r.fim, r.cliente, r.placa || "",
    r.servico, r.status, etapas[r.lavagemStatus] || "Aguardando",
    (r.precoCentavos / 100).toFixed(2).replace(".", ","),
  ]);
  return "\uFEFF" + [cabecalho, ...linhas].map(l => l.map(campoCsv).join(";")).join("\r\n");
}

export function exportarAgenda(data: string, reservas: Reserva[]) {
  const arquivo = new Blob([montarCsv(reservas)], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(arquivo);
  const link = document.createElement("a");
  link.href = url;
  link.download = `agenda-${data}.csv`;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
