const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { createRequire } = require('node:module');

for (const [app, area] of [['front', 'admin'], ['front', 'usuario'], ['frontc', 'usuario']]) {
  test(`${app}/${area} encaminha apenas o cabeçalho de IP configurado`, async () => {
    const filename = path.resolve(app, 'my-app/app/api', area, '[...acao]/route.ts');
    const requireApp = createRequire(filename);
    const ts = requireApp('typescript');
    const { NextRequest } = requireApp('next/server');
    const compiled = ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    }).outputText;
    const module = { exports: {} };
    const run = vm.runInThisContext(`(function(require,module,exports){${compiled}\n})`, { filename });
    run(requireApp, module, module.exports);
    const originalFetch = global.fetch;
    const originalToken = process.env.GATEWAY_TOKEN;
    const originalHeader = process.env.TRUSTED_CLIENT_IP_HEADER;
    process.env.GATEWAY_TOKEN = 'test-only-gateway-token';
    try {
      for (const [configured, value, expected] of [
        ['X-Real-IP', '198.51.100.7', '198.51.100.7'],
        ['X-Real-IP', '2001:db8::1', '2001:db8::1'],
        ['X-Real-IP', '198.51.100.7,203.0.113.8', null],
        ['', '198.51.100.7', null],
      ]) {
        process.env.TRUSTED_CLIENT_IP_HEADER = configured;
        let forwarded = false;
        global.fetch = async (_url, options) => {
          forwarded = true;
          assert.equal(options.headers.get('X-Client-IP'), expected);
          assert.equal(options.headers.get('X-Gateway-Token'), 'test-only-gateway-token');
          return new Response('[]', { headers: { 'Content-Type': 'application/json' } });
        };
        const request = new NextRequest(`https://test.invalid/api/${area}/servicos`, {
          headers: { 'X-Real-IP': value, 'X-Client-IP': '203.0.113.99',
            'X-Forwarded-For': '203.0.113.99', Cookie: 'admin_token=test-only; token=test-only' },
        });
        const response = await module.exports.GET(request, { params: Promise.resolve({ acao: ['servicos'] }) });
        assert.equal(response.status, 200);
        assert.equal(forwarded, true);
      }
    } finally {
      global.fetch = originalFetch;
      for (const [key, value] of [['GATEWAY_TOKEN', originalToken], ['TRUSTED_CLIENT_IP_HEADER', originalHeader]]) {
        if (value === undefined) delete process.env[key];
        else process.env[key] = value;
      }
    }
  });
}
