import { networkInterfaces } from "node:os";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const ips = Object.values(networkInterfaces()).flat()
  .filter(item => item && item.family === "IPv4" && !item.internal)
  .map(item => item.address);
if (!ips.length) throw new Error("Conecte o computador à rede Wi-Fi ou Ethernet.");

const processos = [];
function encerrar(codigo) {
  for (const processo of processos) processo.kill("SIGTERM");
  process.exitCode = codigo;
}
for (const [pasta, nome, porta] of [["front", "ADM", 3100], ["frontc", "Cliente", 3000]]) {
  const cwd = fileURLToPath(new URL("../" + pasta + "/my-app/", import.meta.url));
  for (const ip of ips) console.log(nome + " no celular: http://" + ip + ":" + porta);
  const processo = spawn(process.execPath, ["node_modules/next/dist/bin/next", "dev", "-H", "0.0.0.0", "-p", String(porta)], {
    cwd, stdio: "inherit", env: { ...process.env, LAVA_REDE_IPS: ips.join(","), LAVA_REDE_PORTA: String(porta) }
  });
  processos.push(processo);
  processo.once("exit", codigo => { if (codigo) encerrar(codigo); });
}
process.once("SIGINT", () => encerrar(0));
process.once("SIGTERM", () => encerrar(0));
