export function fecharModal(elemento: HTMLElement, fechar: () => void) {
  const fundo = elemento.closest<HTMLElement>(".modal-fundo");
  if (!fundo || window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
    fechar();
    return;
  }
  if (fundo.classList.contains("modal-saindo")) return;
  fundo.classList.add("modal-saindo");
  window.setTimeout(fechar, 240);
}
