const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const { createRequire } = require("node:module");
const path = require("node:path");

for (const app of ["front", "frontc"]) {
  test(app + " pausa consultas, retoma e descarta respostas antigas", async () => {
    const arquivo = path.resolve(app, "my-app/features/acompanhar.ts");
    const ts = createRequire(arquivo)("typescript");
    const codigo = ts.transpileModule(fs.readFileSync(arquivo, "utf8"), {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    }).outputText;
    const window = new EventTarget();
    const document = new EventTarget();
    const timers = new Map();
    let id = 0;
    window.setTimeout = callback => { timers.set(++id, callback); return id; };
    window.clearTimeout = chave => timers.delete(chave);
    const exports = {};
    vm.runInNewContext(codigo, { window, document, exports, AbortController });
    const pedidos = [], dados = [], erros = [];
    const parar = exports.acompanhar(signal => new Promise((resolve, reject) =>
      pedidos.push({ signal, resolve, reject })), valor => dados.push(valor), e => erros.push(e));
    window.dispatchEvent(new Event("online"));
    assert.equal(pedidos.length, 1);
    document.hidden = true;
    document.dispatchEvent(new Event("visibilitychange"));
    assert.ok(pedidos[0].signal.aborted);
    document.hidden = false;
    document.dispatchEvent(new Event("visibilitychange"));
    pedidos[0].resolve("antigo");
    pedidos[1].resolve("agendamento novo");
    await new Promise(setImmediate);
    assert.deepEqual(dados, ["agendamento novo"]);
    assert.equal(timers.size, 1);
    [...timers.values()][0]();
    pedidos[2].reject(new Error("rede indisponível"));
    await new Promise(setImmediate);
    assert.equal(erros.length, 1);
    assert.equal(timers.size, 1);
    window.dispatchEvent(new Event("online"));
    pedidos[3].resolve("lavagem confirmada");
    await new Promise(setImmediate);
    assert.deepEqual(dados, ["agendamento novo", "lavagem confirmada"]);
    parar();
    window.dispatchEvent(new Event("online"));
    assert.equal(timers.size, 0);
    assert.equal(pedidos.length, 4);
  });
}
