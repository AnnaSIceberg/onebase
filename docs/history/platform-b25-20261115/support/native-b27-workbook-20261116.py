import base64
import hashlib
import json
import os
import pathlib
import secrets
import socket
import subprocess
import time
import urllib.request

import openpyxl
from playwright.sync_api import sync_playwright

B = pathlib.Path(__file__).parent
M = json.loads((B / 'b27-workbook-build-20261116/manifest.json').read_text('utf-8'))
assert M['status'] == 'BUILD_AND_TARGETED_TESTS_PASS_SDK_READY_NATIVE_PENDING'
SDK = pathlib.Path(M['artifacts']['windows']['path'])
with SDK.open('rb') as stream:
    assert hashlib.file_digest(stream, 'sha256').hexdigest() == M['artifacts']['windows']['sha256']
W = B / 'b27-native-workbook-attempt1-20261116'
W.mkdir(exist_ok=False)
P, DB, PORT = W / 'config', W / 'own-native.sqlite', 18582
ORIGIN = f'http://127.0.0.1:{PORT}'
LOGIN, PASSWORD = 'own_b27_admin', 'B27!' + secrets.token_urlsafe(24)
ENV = dict(os.environ, GOMAXPROCS='2')
for key in ['DATABASE_URL', 'TEST_DATABASE_URL']:
    ENV.pop(key, None)
server = log = driver = browser = context = None
errors = []
receipt = {'status': 'RUNNING', 'runtime': M['runtime'], 'source_t8': M['source_t8'], 'base_b26': M['base_b26'], 'sdk_windows_sha256': M['artifacts']['windows']['sha256'], 'checks': {}, 'cleanup': {}, 'shared_changes': False, 'db_copies': 0, 'go_compiler_started': False, 'request_boundary': 'own exact loopback browser-context allowlist; no OS network-isolation claim'}

def require(name, value):
    receipt['checks'][name] = bool(value)
    if not value:
        raise AssertionError(name)

def write(name, content):
    path = P / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding='utf-8')

def cli(args, data=None):
    result = subprocess.run([str(SDK), *args], env=ENV, input=data, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, creationflags=subprocess.CREATE_NO_WINDOW, timeout=60)
	
    commands = receipt.setdefault('cli_commands', [])
    path = W / ('cli-' + str(len(commands)) + '.log')
    path.write_bytes(result.stdout)
    commands.append({'args': args, 'exit': result.returncode, 'log': str(path), 'sha256': hashlib.sha256(result.stdout).hexdigest()})
    if result.returncode:
        raise RuntimeError('own fixture CLI setup failed')

try:
    with socket.socket() as port:
        port.bind(('127.0.0.1', PORT))
    write('config/app.yaml', "name: B27-workbook-probe\nversion: '1.0'\n")
    write('reports/T8WorkbookProbe.yaml', 'name: T8WorkbookProbe\nresult_proc: Build\nparams:\n - {name: Marker, type: string, default: probe}\n')
    write('src/T8WorkbookProbe.rep.os', '''Функция Build()
Книга=Новый КнигаТабличныхДокументов;Д=Новый ТабличныйДокумент;
Д.Ячейка(1,1).Значение=111;Д.Ячейка(1,1).ЦветФона="#ff0000";
Д.Ячейка(2,1).Текст="first";Д.Объединить(1,1,1,2);
Д.ШиринаКолонки(1,145);Д.ВысотаСтроки(1,24);
Д.ОриентацияСтраницы="landscape";Д.АвтоМасштаб=Истина;
Книга.ДобавитьЛист("СМ",Д);
Д.Очистить();Д.Ячейка(1,1).Значение=222;Д.Ячейка(1,1).ЦветФона="#00ff00";
Д.Ячейка(2,1).Текст="second";Д.Объединить(1,1,1,2);
Д.ШиринаКолонки(1,225);Д.ВысотаСтроки(1,32);Д.ОриентацияСтраницы="portrait";
Книга.ДобавитьЛист("ХД",Д);Д.Ячейка(1,1).Значение=999;
Содержимое=Книга.Записать("native-probe.xlsx","xlsx");
Результат=Новый ТабличныйДокумент;Позиция=1;НомерСтроки=1;
Пока Позиция<=СтрДлина(Содержимое) Цикл
Результат.Ячейка(НомерСтроки,1).Текст="WORKBOOK_CHUNK:"+Сред(Содержимое,Позиция,512);
Позиция=Позиция+512;НомерСтроки=НомерСтроки+1;
КонецЦикла;Возврат Результат;КонецФункции
''')
    write('reports/T8WorkbookEmpty.yaml', 'name: T8WorkbookEmpty\nresult_proc: Build\n')
    write('src/T8WorkbookEmpty.rep.os', '''Функция Build()
Книга=Новый SpreadsheetWorkbook;Содержимое=Книга.Write("empty.xlsx");
Д=Новый ТабличныйДокумент;Д.Ячейка(1,1).Текст="SHOULD_NOT_RETURN";Возврат Д;
КонецФункции''')
    cli(['check', '--project', str(P)])
    require('public_check', True)
    cli(['migrate', '--project', str(P), '--sqlite', str(DB)])
    cli(['user', 'add', LOGIN, '--admin', '--password-stdin', '--project', str(P), '--sqlite', str(DB)], (PASSWORD + '\n').encode())
    log = (W / 'server.log').open('wb')
    server = subprocess.Popen([str(SDK), 'run', '--project', str(P), '--sqlite', str(DB), '--host', '127.0.0.1', '--port', str(PORT)], env=ENV, stdout=log, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, creationflags=subprocess.CREATE_NO_WINDOW)
    for _ in range(120):
        if server.poll() is not None:
            raise RuntimeError('own server exited')
        try:
            with urllib.request.urlopen(ORIGIN + '/login', timeout=.5) as response:
                if response.status == 200:
                    break
        except Exception:
            time.sleep(.1)
    else:
        raise TimeoutError('own server readiness')
    driver = sync_playwright().start()
    browser = driver.chromium.launch(executable_path='C:/Users/a.sevrjukova/AppData/Local/ms-playwright/chromium-1041/chrome-win/chrome.exe', headless=True)
    context = browser.new_context(viewport={'width': 1440, 'height': 900})
    context.route('**/*', lambda route: route.continue_() if route.request.url.startswith(ORIGIN + '/') else route.abort())
    page = context.new_page()
    page.on('pageerror', lambda error: errors.append(type(error).__name__))
    page.goto(ORIGIN + '/login', wait_until='domcontentloaded')
    page.locator('input[name="login"]').fill(LOGIN)
    page.locator('input[name="password"]').fill(PASSWORD)
    page.locator('button[type="submit"]').click()
    page.goto(ORIGIN + '/ui/report/T8WorkbookProbe', wait_until='domcontentloaded')
    with page.expect_response(lambda r: r.request.method == 'POST' and '/ui/report/T8WorkbookProbe' in r.url, timeout=20000) as pending:
        page.get_by_role('button', name='Сформировать', exact=True).click()
    require('native_post303', pending.value.status == 303)
    page.wait_for_url('**/ui/tabular/**', timeout=15000)
    chunks = [text.strip().removeprefix('WORKBOOK_CHUNK:') for text in page.locator('td').all_text_contents() if text.strip().startswith('WORKBOOK_CHUNK:')]
    require('native_result_proc_registered_factory', bool(chunks))
    raw = base64.b64decode(''.join(chunks), validate=True)
    artifact = W / 'native-workbook.xlsx'
    artifact.write_bytes(raw)
    for read_only in [False, True]:
        book = openpyxl.load_workbook(artifact, read_only=read_only, data_only=True)
        mode = 'default_read_only' if read_only else 'normal'
        require(mode + '_sheet_order', book.sheetnames == ['СМ', 'ХД'])
        for name, number, color, orientation, width, height in [('СМ', 111, 'FF0000', 'landscape', 20, 18), ('ХД', 222, '00FF00', 'portrait', (225-5)/7, 24)]:
            sh = book[name]
            cell = sh['A1']
            require(mode + '_' + name + '_typed_value', cell.value == number and type(cell.value) in (int, float))
            require(mode + '_' + name + '_fill', cell.fill.fgColor.rgb[-6:].upper() == color)
            require(mode + '_' + name + '_dimension', sh.calculate_dimension() == 'A1:B2')
            if not read_only:
                require(name + '_merge', str(next(iter(sh.merged_cells.ranges))) == 'A1:B1')
                require(name + '_width', abs(sh.column_dimensions['A'].width - width) < .01)
                require(name + '_height', abs(sh.row_dimensions[1].height - height) < .01)
                require(name + '_page', sh.page_setup.orientation == orientation and sh.page_setup.fitToWidth == 1)
        book.close()
    response = page.goto(ORIGIN + '/ui/report/T8WorkbookEmpty?__run=1', wait_until='domcontentloaded')
    require('native_empty_book500', response.status == 500)
    require('native_empty_book_explicit_error', 'at least one sheet' in page.locator('body').inner_text())
    require('JS0', not errors)
    receipt.update({'status': 'PASS', 'browser_version': browser.version, 'reader_version': openpyxl.__version__, 'artifact': str(artifact), 'artifact_sha256': hashlib.sha256(raw).hexdigest(), 'reset_dimensions': False, 'forced_bounds': False})
except Exception as error:
    import traceback
    receipt.update({'status': 'FAIL', 'failure_type': type(error).__name__, 'failed_check': str(error) if isinstance(error, AssertionError) else None, 'failure_line': traceback.extract_tb(error.__traceback__)[-1].lineno})
finally:
    for key, obj in [('context', context), ('browser', browser)]:
        try:
            if obj:
                obj.close()
            receipt['cleanup'][key] = True
        except Exception:
            receipt['cleanup'][key] = False
    try:
        if driver:
            driver.stop()
        receipt['cleanup']['driver'] = True
    except Exception:
        receipt['cleanup']['driver'] = False
    if server:
        server.terminate()
        try:
            server.wait(timeout=8)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait(timeout=3)
    receipt['cleanup']['app'] = server is None or server.poll() is not None
    if log:
        log.close()
    try:
        with socket.create_connection(('127.0.0.1', PORT), timeout=.2):
            receipt['cleanup']['port_closed'] = False
    except OSError:
        receipt['cleanup']['port_closed'] = True
    if receipt['cleanup']['app']:
        for path in [DB, pathlib.Path(str(DB) + '-wal'), pathlib.Path(str(DB) + '-shm')]:
            assert path.resolve().is_relative_to(W.resolve())
            path.unlink(missing_ok=True)
    receipt['cleanup']['own_auth_db_removed'] = not DB.exists()
    receipt['page_errors'] = len(errors)
    (W / 'receipt.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'status': receipt['status'], 'checks': len(receipt['checks']), 'cleanup': receipt['cleanup'], 'failed_check': receipt.get('failed_check'), 'failure_type': receipt.get('failure_type')}))
    if receipt['status'] != 'PASS' or not all(receipt['cleanup'].values()):
        raise SystemExit(1)
