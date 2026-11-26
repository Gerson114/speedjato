// Cada consulta termina antes de agendar a próxima. Uma aba reaberta consulta imediatamente.
export function acompanhar<T>(
  consultar: (signal: AbortSignal) => Promise<T>,
  receber: (dados: T) => void,
  falhar: (erro: Error) => void,
  intervalo = 5000,
) {
  let timer: number | undefined;
  let atual: AbortController | undefined;
  let encerrado = false;

  async function atualizar() {
    if (encerrado || document.hidden || atual) return;
    window.clearTimeout(timer);
    const pedido = new AbortController();
    atual = pedido;
    try {
      const dados = await consultar(pedido.signal);
      if (!pedido.signal.aborted) receber(dados);
    } catch (erro) {
      if (!pedido.signal.aborted) falhar(erro as Error);
    } finally {
      if (atual === pedido) {
        atual = undefined;
        if (!encerrado && !document.hidden) timer = window.setTimeout(atualizar, intervalo);
      }
    }
  }
  function parar() {
    window.clearTimeout(timer);
    atual?.abort();
    atual = undefined;
  }
  function visibilidade() {
    if (document.hidden) parar();
    else void atualizar();
  }
  document.addEventListener("visibilitychange", visibilidade);
  window.addEventListener("online", visibilidade);
  void atualizar();
  return () => {
    encerrado = true;
    parar();
    document.removeEventListener("visibilitychange", visibilidade);
    window.removeEventListener("online", visibilidade);
  };
}
